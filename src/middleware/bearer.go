package middleware

import "strings"

// AuthenticatedSubjectKey ключ пользователя в gin.Context.
const AuthenticatedSubjectKey = "authenticated_subject"

// ResourceAccess описывает разрешённые операции с ресурсом Registry.
type ResourceAccess struct {
	Type    string
	Name    string
	Actions []string
}

// Identity содержит проверенные данные аутентифицированного пользователя.
type Identity struct {
	Subject string
	Access  []ResourceAccess
}

// TokenValidator проверяет Bearer-токен и возвращает пользователя.
type TokenValidator func(token string) (Identity, error)

/*
bearerToken извлекает Bearer-токен из заголовка Authorization.

	header - значение заголовка Authorization.
*/
func bearerToken(header string) (string, bool) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	if parts[1] == "" {
		return "", false
	}

	return parts[1], true
}
