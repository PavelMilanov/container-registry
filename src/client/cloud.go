package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

/*
GetCloudList делает запрос к API для получения списка пространств.

Returns:

	[]string: список пространств.
	error: ошибка при выполнении запроса.
*/
func (c *Client) GetCloudList(auth string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.ServerURL+"/api/cloud/list", nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{}
	req.Header.Add("Authorization", "Bearer "+auth)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(string(body))
	}
	var data struct {
		Clouds []string `json:"clouds"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	return data.Clouds, nil
}

/*
AddCloud делает запрос к API для создания пространства.

Params:

	auth - токен авторизации.
	cloud - название пространства.

Returns:

	error - ошибка, если запрос не удался.
*/
func (c *Client) AddCloud(auth string, cloud string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.ServerURL+"/api/cloud", nil)
	if err != nil {
		return err
	}
	client := &http.Client{}
	req.Header.Add("Authorization", "Bearer "+auth)
	req.Header.Add("Content-Type", "application/json")
	req.Body = io.NopCloser(strings.NewReader(fmt.Sprintf(`{"cloud": "%s"}`, cloud)))
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusCreated {
		return errors.New(string(body))
	}
	var data struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return err
	}
	fmt.Println(data.Message)
	return nil
}

func (c *Client) DelCloud(auth string, cloud string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.ServerURL+"/api/cloud", nil)
	if err != nil {
		return err
	}
	client := &http.Client{}
	req.Header.Add("Authorization", "Bearer "+auth)
	req.Header.Add("Content-Type", "application/json")
	req.Body = io.NopCloser(strings.NewReader(fmt.Sprintf(`{"cloud": "%s"}`, cloud)))
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusAccepted {
		return errors.New(string(body))
	}
	var data struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return err
	}
	fmt.Println(data.Message)
	return nil
}

func (c *Client) GetRepositoriesList(auth, cloud string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/cloud/%s/repositories", c.ServerURL, cloud), nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{}
	req.Header.Add("Authorization", "Bearer "+auth)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(string(body))
	}
	var result struct {
		Repositories []string `json:"repositories"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result.Repositories, nil
}
