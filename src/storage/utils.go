package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"uuid"

	"github.com/PavelMilanov/container-registry/config"
)

/*
validNamespaceName проверяет, что имя пространства является одним сегментом пути.

	name - имя пространства.
*/
func validNamespaceName(name string) bool {
	return name != "" &&
		name != "." &&
		name != ".." &&
		!strings.ContainsAny(name, `/\\`)
}

/*
localUploadPath формирует путь к временному файлу загрузки Blob.

	uploadID - идентификатор загрузки.

Перед формированием пути проверяет, что uploadID является корректным UUID.
Возвращает ErrUploadNotFound при некорректном идентификаторе.
*/
func localUploadPath(uploadID string) (string, error) {
	if _, err := uuid.Parse(uploadID); err != nil {
		return "", ErrUploadNotFound
	}

	return filepath.Join(config.TMP_PATH, uploadID), nil
}

/*
copyWithContext копирует данные из источника в получатель с поддержкой отмены.

	ctx - контекст выполнения операции.
	dst - получатель данных.
	src - источник данных.

Данные копируются частями без полной загрузки содержимого в память.
Возвращает количество фактически записанных байт.
*/
func copyWithContext(
	ctx context.Context,
	dst io.Writer,
	src io.Reader,
) (int64, error) {
	buffer := make([]byte, 256*1024)
	var total int64

	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}

		n, readErr := src.Read(buffer)
		if n > 0 {
			written, writeErr := dst.Write(buffer[:n])
			total += int64(written)

			if writeErr != nil {
				return total, writeErr
			}
			if written != n {
				return total, io.ErrShortWrite
			}
		}

		if readErr == io.EOF {
			return total, nil
		}
		if readErr != nil {
			return total, readErr
		}
	}
}

/*
parseSHA256Digest проверяет digest и извлекает его шестнадцатеричную часть.

	digest - контрольная сумма в формате sha256:<hex>.

Возвращает SHA-256 без префикса "sha256:".
При неверном алгоритме, длине или формате возвращает ErrInvalidDigest.
*/
func parseSHA256Digest(digest string) (string, error) {
	algorithm, encoded, ok := strings.Cut(digest, ":")
	if !ok || algorithm != "sha256" {
		return "", ErrInvalidDigest
	}

	if len(encoded) != 64 {
		return "", ErrInvalidDigest
	}
	if encoded != strings.ToLower(encoded) {
		return "", ErrInvalidDigest
	}

	if _, err := hex.DecodeString(encoded); err != nil {
		return "", ErrInvalidDigest
	}

	return encoded, nil
}

/*
calculateFileDigest вычисляет контрольную сумму SHA-256 содержимого файла.

	ctx - контекст выполнения операции.
	path - путь к файлу.

Файл читается потоково через copyWithContext.
Возвращает контрольную сумму в формате sha256:<hex>.
*/
func calculateFileDigest(
	ctx context.Context,
	path string,
) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := copyWithContext(ctx, hasher, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("sha256:%x", hasher.Sum(nil)), nil
}
