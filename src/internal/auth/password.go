// Package auth реализует хеширование паролей и работу с JWT приложения.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	defaultArgonMemory      = 19 * 1024
	defaultArgonIterations  = 2
	defaultArgonParallelism = 1
	defaultArgonSaltLength  = 16
	defaultArgonKeyLength   = 32
	maxPasswordBytes        = 1024
	maxArgonMemory          = 256 * 1024
	maxArgonIterations      = 10
	maxArgonParallelism     = 16
)

var (
	ErrEmptyPassword       = errors.New("password is empty")
	ErrPasswordTooLong     = errors.New("password is too long")
	ErrInvalidPasswordHash = errors.New("invalid password hash")
)

// PasswordHasher безопасно хеширует и проверяет пароли.
type PasswordHasher struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	saltLength  uint32
	keyLength   uint32
}

/*
NewPasswordHasher создаёт PasswordHasher с параметрами Argon2id приложения.
*/
func NewPasswordHasher() *PasswordHasher {
	return &PasswordHasher{
		memory:      defaultArgonMemory,
		iterations:  defaultArgonIterations,
		parallelism: defaultArgonParallelism,
		saltLength:  defaultArgonSaltLength,
		keyLength:   defaultArgonKeyLength,
	}
}

/*
Hash создаёт Argon2id-хеш пароля в PHC-формате.

	password - пароль пользователя.
*/
func (h *PasswordHasher) Hash(password string) (string, error) {
	if err := validatePassword(password); err != nil {
		return "", err
	}

	salt := make([]byte, h.saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("не удалось создать соль пароля: %w", err)
	}

	key := argon2.IDKey(
		[]byte(password),
		salt,
		h.iterations,
		h.memory,
		h.parallelism,
		h.keyLength,
	)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		h.memory,
		h.iterations,
		h.parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

/*
Verify проверяет пароль по сохранённому Argon2id-хешу.

	encodedHash - хеш пароля в PHC-формате.
	password - пароль пользователя.

Возвращает needsUpgrade=true, если параметры хеша необходимо обновить.
*/
func (h *PasswordHasher) Verify(
	encodedHash string,
	password string,
) (valid bool, needsUpgrade bool, err error) {
	if err := validatePassword(password); err != nil {
		return false, false, err
	}

	params, salt, expected, err := parseArgon2idHash(encodedHash)
	if err != nil {
		return false, false, err
	}
	actual := argon2.IDKey(
		[]byte(password),
		salt,
		params.iterations,
		params.memory,
		params.parallelism,
		uint32(len(expected)),
	)
	valid = subtle.ConstantTimeCompare(actual, expected) == 1
	needsUpgrade = valid && (params.memory != h.memory ||
		params.iterations != h.iterations ||
		params.parallelism != h.parallelism ||
		uint32(len(salt)) != h.saltLength ||
		uint32(len(expected)) != h.keyLength)
	return valid, needsUpgrade, nil
}

type argon2Params struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
}

/*
parseArgon2idHash разбирает и проверяет Argon2id-хеш в PHC-формате.

	encodedHash - сохранённый хеш пароля.
*/
func parseArgon2idHash(
	encodedHash string,
) (argon2Params, []byte, []byte, error) {
	var params argon2Params
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return params, nil, nil, ErrInvalidPasswordHash
	}

	if !strings.HasPrefix(parts[2], "v=") {
		return params, nil, nil, ErrInvalidPasswordHash
	}
	version, err := strconv.Atoi(strings.TrimPrefix(parts[2], "v="))
	if err != nil || version != argon2.Version {
		return params, nil, nil, ErrInvalidPasswordHash
	}
	count, err := fmt.Sscanf(
		parts[3],
		"m=%d,t=%d,p=%d",
		&params.memory,
		&params.iterations,
		&params.parallelism,
	)
	if err != nil || count != 3 || parts[3] != fmt.Sprintf(
		"m=%d,t=%d,p=%d",
		params.memory,
		params.iterations,
		params.parallelism,
	) {
		return params, nil, nil, ErrInvalidPasswordHash
	}
	if params.memory == 0 || params.memory > maxArgonMemory ||
		params.iterations == 0 || params.iterations > maxArgonIterations ||
		params.parallelism == 0 || params.parallelism > maxArgonParallelism {
		return params, nil, nil, ErrInvalidPasswordHash
	}

	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil || len(salt) < 8 || len(salt) > 64 {
		return params, nil, nil, ErrInvalidPasswordHash
	}
	expected, err := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil || len(expected) < 16 || len(expected) > 64 {
		return params, nil, nil, ErrInvalidPasswordHash
	}

	return params, salt, expected, nil
}

/*
validatePassword проверяет допустимый размер пароля.

	password - пароль пользователя.
*/
func validatePassword(password string) error {
	if password == "" {
		return ErrEmptyPassword
	}
	if len(password) > maxPasswordBytes {
		return ErrPasswordTooLong
	}
	return nil
}
