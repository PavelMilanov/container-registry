package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/PavelMilanov/container-registry/db"
	registryauth "github.com/PavelMilanov/container-registry/internal/auth"
	"github.com/sirupsen/logrus"
)

var ErrInvalidCredentials = errors.New("неверные логин или пароль")

// AuthService координирует регистрацию, вход и выпуск JWT.
type AuthService struct {
	users     UserRepository
	passwords PasswordManager
	tokens    TokenManager
}

// UserRepository предоставляет AuthService операции с пользователями.
type UserRepository interface {
	Create(ctx context.Context, user *db.User) error
	FindByName(ctx context.Context, name string) (db.User, error)
	UpdatePassword(ctx context.Context, userID int, passwordHash string) error
}

// PasswordManager хеширует и проверяет пароли пользователей.
type PasswordManager interface {
	Hash(password string) (string, error)
	Verify(hash, password string) (valid bool, needsUpgrade bool, err error)
}

// TokenManager выпускает и проверяет JWT пользователей.
type TokenManager interface {
	Issue(
		subject string,
		access []registryauth.ResourceAction,
	) (registryauth.IssuedToken, error)
	Validate(rawToken string) (registryauth.Claims, error)
}

/*
NewAuthService создаёт сервис аутентификации.

	users - repository пользователей.
	passwords - реализация безопасного хеширования паролей.
	tokens - менеджер выпуска и проверки JWT.
*/
func NewAuthService(
	users UserRepository,
	passwords PasswordManager,
	tokens TokenManager,
) *AuthService {
	return &AuthService{
		users:     users,
		passwords: passwords,
		tokens:    tokens,
	}
}

/*
EnsureUser создаёт пользователя, если он отсутствует.

	ctx - контекст выполнения операции.
	username - имя пользователя.
	password - пароль пользователя.
*/
func (s *AuthService) EnsureUser(
	ctx context.Context,
	username string,
	password string,
) error {
	if err := s.validateDependencies(); err != nil {
		return err
	}
	if _, err := s.users.FindByName(ctx, username); err == nil {
		return nil
	} else if !errors.Is(err, db.ErrUserNotFound) {
		return err
	}

	return s.Register(ctx, username, password)
}

/*
Register регистрирует пользователя с безопасным хешем пароля.

	ctx - контекст выполнения операции.
	username - имя пользователя.
	password - пароль пользователя.
*/
func (s *AuthService) Register(
	ctx context.Context,
	username string,
	password string,
) error {
	if err := s.validateDependencies(); err != nil {
		return err
	}
	if username == "" {
		return ErrInvalidCredentials
	}

	passwordHash, err := s.passwords.Hash(password)
	if err != nil {
		return err
	}
	user := db.User{Name: username, Password: passwordHash}
	if err := s.users.Create(ctx, &user); err != nil {
		return err
	}
	return nil
}

/*
Login проверяет учётные данные и выпускает JWT.

	ctx - контекст выполнения операции.
	username - имя пользователя.
	password - пароль пользователя.
	access - разрешения Docker Registry, включаемые в JWT.
*/
func (s *AuthService) Login(
	ctx context.Context,
	username string,
	password string,
	access []registryauth.ResourceAction,
) (registryauth.IssuedToken, error) {
	var result registryauth.IssuedToken
	if err := s.validateDependencies(); err != nil {
		return result, err
	}

	user, err := s.users.FindByName(ctx, username)
	if errors.Is(err, db.ErrUserNotFound) {
		return result, ErrInvalidCredentials
	}
	if err != nil {
		return result, err
	}

	valid, needsUpgrade, err := s.passwords.Verify(user.Password, password)
	if err != nil {
		return result, err
	}
	if !valid {
		return result, ErrInvalidCredentials
	}
	if needsUpgrade {
		passwordHash, err := s.passwords.Hash(password)
		if err != nil {
			return result, err
		}
		if err := s.users.UpdatePassword(ctx, user.ID, passwordHash); err != nil {
			return result, err
		}
	}

	token, err := s.tokens.Issue(user.Name, access)
	if err != nil {
		return result, err
	}
	logrus.WithField("username", username).Info("Успешная авторизация")
	return token, nil
}

/*
ValidateToken проверяет JWT и возвращает его claims.

	rawToken - подписанный JWT.
*/
func (s *AuthService) ValidateToken(
	rawToken string,
) (registryauth.Claims, error) {
	if err := s.validateDependencies(); err != nil {
		return registryauth.Claims{}, err
	}
	return s.tokens.Validate(rawToken)
}

/*
validateDependencies проверяет зависимости сервиса аутентификации.
*/
func (s *AuthService) validateDependencies() error {
	if s == nil || s.users == nil || s.passwords == nil || s.tokens == nil {
		return fmt.Errorf("auth service is not configured")
	}
	return nil
}
