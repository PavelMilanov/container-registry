package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const minimumHMACKeyLength = 32

var (
	ErrInvalidTokenConfig = errors.New("invalid token configuration")
	ErrInvalidToken       = errors.New("invalid token")
)

// TokenConfig содержит параметры выпуска и проверки JWT.
type TokenConfig struct {
	Secret   []byte
	Issuer   string
	Audience string
	TTL      time.Duration
}

// ResourceAction описывает доступ JWT к ресурсу Docker Registry.
type ResourceAction struct {
	Type    string   `json:"type"`
	Name    string   `json:"name"`
	Actions []string `json:"actions"`
}

// Claims содержит проверяемые claims JWT приложения.
type Claims struct {
	Access []ResourceAction `json:"access,omitempty"`
	jwt.RegisteredClaims
}

// IssuedToken содержит JWT и рассчитанные временные границы его действия.
type IssuedToken struct {
	Value     string
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// TokenManager выпускает и проверяет JWT приложения.
type TokenManager struct {
	secret   []byte
	issuer   string
	audience string
	ttl      time.Duration
	now      func() time.Time
}

/*
NewTokenManager создаёт TokenManager с проверенной конфигурацией.

	config - параметры подписи и стандартных JWT claims.
*/
func NewTokenManager(config TokenConfig) (*TokenManager, error) {
	if len(config.Secret) < minimumHMACKeyLength {
		return nil, fmt.Errorf(
			"%w: JWT secret должен содержать не менее %d байт",
			ErrInvalidTokenConfig,
			minimumHMACKeyLength,
		)
	}
	if config.Issuer == "" {
		return nil, fmt.Errorf("%w: JWT issuer не указан", ErrInvalidTokenConfig)
	}
	if config.Audience == "" {
		return nil, fmt.Errorf("%w: JWT audience не указан", ErrInvalidTokenConfig)
	}
	if config.TTL <= 0 {
		return nil, fmt.Errorf("%w: JWT TTL должен быть больше нуля", ErrInvalidTokenConfig)
	}

	return &TokenManager{
		secret:   append([]byte(nil), config.Secret...),
		issuer:   config.Issuer,
		audience: config.Audience,
		ttl:      config.TTL,
		now:      time.Now,
	}, nil
}

/*
Issue выпускает JWT для пользователя.

	subject - идентификатор пользователя.
	access - разрешения Docker Registry.
*/
func (m *TokenManager) Issue(
	subject string,
	access []ResourceAction,
) (IssuedToken, error) {
	var result IssuedToken
	if subject == "" {
		return result, fmt.Errorf("%w: JWT subject не указан", ErrInvalidToken)
	}

	issuedAt := m.now().UTC().Truncate(time.Second)
	expiresAt := issuedAt.Add(m.ttl)
	claims := Claims{
		Access: append([]ResourceAction(nil), access...),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   subject,
			Audience:  jwt.ClaimStrings{m.audience},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			NotBefore: jwt.NewNumericDate(issuedAt),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	value, err := token.SignedString(m.secret)
	if err != nil {
		return result, fmt.Errorf("не удалось подписать JWT: %w", err)
	}

	return IssuedToken{
		Value:     value,
		IssuedAt:  issuedAt,
		ExpiresAt: expiresAt,
	}, nil
}

/*
Validate проверяет JWT и возвращает его claims.

	rawToken - подписанный JWT.
*/
func (m *TokenManager) Validate(rawToken string) (Claims, error) {
	var claims Claims
	if rawToken == "" {
		return claims, ErrInvalidToken
	}

	token, err := jwt.ParseWithClaims(
		rawToken,
		&claims,
		func(token *jwt.Token) (any, error) {
			return m.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(m.issuer),
		jwt.WithAudience(m.audience),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithTimeFunc(m.now),
	)
	if err != nil {
		return Claims{}, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}
	if !token.Valid || claims.Subject == "" || claims.IssuedAt == nil {
		return Claims{}, ErrInvalidToken
	}

	return claims, nil
}
