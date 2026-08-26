package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/PavelMilanov/container-registry/db"
	registryauth "github.com/PavelMilanov/container-registry/internal/auth"
	"github.com/PavelMilanov/container-registry/services"
)

const tokenTTL = 2 * time.Hour

/*
newAuthService создаёт сервис аутентификации и пользователя по умолчанию.

	ctx - контекст инициализации.
	database - подключение к SQLite.
	env - конфигурация приложения.
*/
func newAuthService(
	ctx context.Context,
	database *db.SQLite,
	env *config.Env,
) (*services.AuthService, error) {
	passwords := registryauth.NewPasswordHasher()
	tokens, err := registryauth.NewTokenManager(registryauth.TokenConfig{
		Secret:   []byte(env.Server.Jwt),
		Issuer:   config.DefaultTokenIssuer,
		Audience: config.DefaultTokenService,
		TTL:      tokenTTL,
	})
	if err != nil {
		return nil, fmt.Errorf("не удалось создать менеджер JWT: %w", err)
	}

	authService := services.NewAuthService(
		db.NewUserRepository(database),
		passwords,
		tokens,
	)
	if err := authService.EnsureUser(
		ctx,
		env.DefaultUser.Login,
		env.DefaultUser.Password,
	); err != nil {
		return nil, fmt.Errorf(
			"не удалось подготовить default_user %q: %w",
			env.DefaultUser.Login,
			err,
		)
	}

	return authService, nil
}
