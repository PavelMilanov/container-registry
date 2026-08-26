package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

/*
Login делает запрос к API для аутентификации и возвращает токен.

Params:

	login - логин пользователя.
	password - пароль пользователя.

Returns:

	string - токен.
	error - ошибка, если запрос не удался.
*/
func (c *Client) Login(ctx context.Context, login, password string) (string, error) {
	data := map[string]string{
		"username": login,
		"password": password,
	}
	jsonData, _ := json.Marshal(data)
	req, cancel, err := c.newRequest(ctx, http.MethodPost, "/login", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	defer cancel()
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", errors.New(string(body))
	}
	var result struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	return result.Token, nil
}
