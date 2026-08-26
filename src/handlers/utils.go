package handlers

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/PavelMilanov/container-registry/storage"
	"github.com/labstack/echo/v5"
)

type contentRange struct {
	start int64
	end   int64
}

/*
size возвращает количество байт в диапазоне Content-Range.
*/
func (r contentRange) size() int64 {
	return r.end - r.start + 1
}

type exactLengthReader struct {
	source    io.Reader
	remaining int64
}

/*
parseContentRange извлекает начальную позицию загрузки из Content-Range.

	value - значение HTTP-заголовка Content-Range.

Поддерживает форматы "<start>-<end>" и "bytes <start>-<end>".
Возвращает обе границы либо ошибку при пустом или некорректном диапазоне.
*/
func parseContentRange(value string) (contentRange, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return contentRange{}, errors.New("Content-Range is required")
	}

	if strings.HasPrefix(value, "bytes ") {
		value = strings.TrimSpace(strings.TrimPrefix(value, "bytes "))
	}

	startText, endText, ok := strings.Cut(value, "-")
	if !ok {
		return contentRange{}, errors.New("invalid Content-Range")
	}

	start, err := strconv.ParseInt(strings.TrimSpace(startText), 10, 64)
	if err != nil || start < 0 {
		return contentRange{}, errors.New("invalid Content-Range start")
	}

	end, err := strconv.ParseInt(strings.TrimSpace(endText), 10, 64)
	if err != nil || end < start {
		return contentRange{}, errors.New("invalid Content-Range end")
	}

	return contentRange{start: start, end: end}, nil
}

/*
newExactLengthReader создаёт поток с проверкой ожидаемого размера.

	source - исходный поток данных.
	expected - ожидаемое количество байт.
*/
func newExactLengthReader(source io.Reader, expected int64) io.Reader {
	return &exactLengthReader{
		source:    source,
		remaining: expected,
	}
}

/*
Read читает данные и проверяет, что размер потока совпадает с ожидаемым.

	p - буфер для прочитанных данных.
*/
func (r *exactLengthReader) Read(p []byte) (int, error) {
	if r.remaining > 0 {
		if int64(len(p)) > r.remaining {
			p = p[:r.remaining]
		}

		n, err := r.source.Read(p)
		r.remaining -= int64(n)
		if err == io.EOF && r.remaining > 0 {
			return n, storage.ErrInvalidLength
		}
		return n, err
	}

	var extra [1]byte
	n, err := r.source.Read(extra[:])
	if n > 0 {
		return 0, storage.ErrInvalidLength
	}
	return 0, err
}

/*
respondBlobUploadError преобразует ошибку BlobUploadStore в HTTP-ответ.

	c - контекст HTTP-запроса Echo.
	err - ошибка операции загрузки Blob.

Типизированные ошибки storage преобразуются в соответствующие HTTP-статусы
и коды ошибок Docker Registry API. Неизвестная ошибка возвращает статус 500.
*/
func respondBlobUploadError(c *echo.Context, err error) error {
	addRequestError(c, err)

	status := http.StatusInternalServerError
	code := "UNKNOWN"
	message := "internal server error"

	switch {
	case errors.Is(err, storage.ErrUploadNotFound):
		status = http.StatusNotFound
		code = "BLOB_UPLOAD_UNKNOWN"
		message = "blob upload not found"
	case errors.Is(err, storage.ErrInvalidOffset):
		status = http.StatusRequestedRangeNotSatisfiable
		code = "RANGE_INVALID"
		message = "invalid upload offset"
	case errors.Is(err, storage.ErrInvalidLength):
		status = http.StatusRequestedRangeNotSatisfiable
		code = "RANGE_INVALID"
		message = "invalid upload length"
	case errors.Is(err, storage.ErrInvalidDigest),
		errors.Is(err, storage.ErrDigestMismatch):
		status = http.StatusBadRequest
		code = "DIGEST_INVALID"
		message = "invalid blob digest"
	}

	return c.JSON(status, map[string]any{
		"errors": []map[string]any{
			{
				"code":    code,
				"message": message,
			},
		},
	})
}
