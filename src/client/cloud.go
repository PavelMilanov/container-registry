package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

/*
GetCloudList делает запрос к API для получения списка пространств.

Returns:

	[]string: список пространств.
	error: ошибка при выполнении запроса.
*/
func (c *Client) GetCloudList(ctx context.Context) ([]string, error) {
	req, cancel, err := c.newAuthorizedRequest(ctx, http.MethodGet, "/api/cloud/list", nil)
	if err != nil {
		return nil, err
	}
	defer cancel()
	resp, err := c.do(req)
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
func (c *Client) AddCloud(ctx context.Context, cloud string) error {
	req, cancel, err := c.newAuthorizedRequest(ctx, http.MethodPost, "/api/cloud/create", strings.NewReader(fmt.Sprintf(`{"cloud": "%s"}`, cloud)))
	if err != nil {
		return err
	}
	defer cancel()
	req.Header.Add("Content-Type", "application/json")
	resp, err := c.do(req)
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
	fmt.Println(string(body))
	return nil
}

func (c *Client) DelCloud(ctx context.Context, cloud string) error {
	req, cancel, err := c.newAuthorizedRequest(ctx, http.MethodDelete, "/api/cloud/delete", strings.NewReader(fmt.Sprintf(`{"cloud": "%s"}`, cloud)))
	if err != nil {
		return err
	}
	defer cancel()
	req.Header.Add("Content-Type", "application/json")
	resp, err := c.do(req)
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
	fmt.Println(string(body))
	return nil
}

func (c *Client) GetRepositoriesList(ctx context.Context, cloud string) ([]string, error) {
	req, cancel, err := c.newAuthorizedRequest(ctx, http.MethodGet, fmt.Sprintf("/api/cloud/%s", cloud), nil)
	if err != nil {
		return nil, err
	}
	defer cancel()
	resp, err := c.do(req)
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
		Repositories []string `json:"repositories"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	return data.Repositories, nil
}

func (c *Client) GetImagesList(ctx context.Context, cloud, repository string) ([]string, error) {
	req, cancel, err := c.newAuthorizedRequest(ctx, http.MethodGet, fmt.Sprintf("/api/cloud/%s/%s", cloud, repository), nil)
	if err != nil {
		return nil, err
	}
	defer cancel()
	resp, err := c.do(req)
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
		Images []string `json:"images"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	return data.Images, nil
}

func (c *Client) DelImage(ctx context.Context, cloud string, repository string, tag string) error {
	req, cancel, err := c.newAuthorizedRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/cloud/%s/%s", cloud, repository), nil)
	if err != nil {
		return err
	}
	defer cancel()
	q := req.URL.Query()
	q.Add("tag", tag)
	req.URL.RawQuery = q.Encode()
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusNoContent {
		return errors.New(string(body))
	}
	fmt.Println(string(body))
	return nil
}

func (c *Client) DelRepository(ctx context.Context, cloud string, repository string) error {
	req, cancel, err := c.newAuthorizedRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/cloud/%s/%s", cloud, repository), nil)
	if err != nil {
		return err
	}
	defer cancel()
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusNoContent {
		return errors.New(string(body))
	}
	fmt.Println(string(body))
	return nil
}
