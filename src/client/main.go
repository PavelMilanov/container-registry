package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const defaultRequestTimeout = 5 * time.Second

type Client struct {
	ServerURL  string
	httpClient *http.Client
	timeout    time.Duration
	token      string
	tokenPath  string
}

func NewClient() *Client {
	return &Client{
		ServerURL:  "http://0.0.0.0:5050",
		httpClient: http.DefaultClient,
		timeout:    defaultRequestTimeout,
		tokenPath:  defaultTokenPath(),
	}
}

func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, context.CancelFunc, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	timeout := c.timeout
	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	req, err := http.NewRequestWithContext(timeoutCtx, method, c.ServerURL+path, body)
	if err != nil {
		cancel()
		return nil, nil, err
	}
	return req, cancel, nil
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	client := c.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	return client.Do(req)
}

func (c *Client) newAuthorizedRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, context.CancelFunc, error) {
	auth, err := c.getToken()
	if err != nil {
		return nil, nil, err
	}
	req, cancel, err := c.newRequest(ctx, method, path, body)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Add("Authorization", "Bearer "+auth)
	return req, cancel, nil
}

func defaultTokenPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil || configDir == "" {
		return ""
	}
	return filepath.Join(configDir, "container-registry", "auth")
}

func (c *Client) HealthCheck(ctx context.Context) error {
	req, cancel, err := c.newRequest(ctx, http.MethodGet, "/check", nil)
	if err != nil {
		return err
	}
	defer cancel()
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("unhealthy")
	}
	return nil
}

func (c *Client) GarbageCollection(ctx context.Context) error {
	req, cancel, err := c.newAuthorizedRequest(ctx, http.MethodPost, "/api/garbage/collection", nil)
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
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		c.removeToken()
		return errors.New("Необходима авторизация")
	case http.StatusAccepted:
		fmt.Println(string(body))
		return nil
	default:
		return errors.New(string(body))
	}
}

func (c *Client) GarbageTags(ctx context.Context) error {
	req, cancel, err := c.newAuthorizedRequest(ctx, http.MethodPost, "/api/garbage/tags", nil)
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
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		c.removeToken()
		return errors.New("Необходима авторизация")
	case http.StatusAccepted:
		fmt.Println(string(body))
		return nil
	default:
		return errors.New(string(body))
	}

}

func (c *Client) getToken() (string, error) {
	if c.token != "" {
		return c.token, nil
	}
	if c.tokenPath == "" {
		return "", errors.New("Доступ запрещен. Необходимо авторизоваться.")
	}
	auth, err := os.ReadFile(c.tokenPath)
	if err != nil {
		return "", errors.New("Доступ запрещен. Необходимо авторизоваться.")
	}
	c.token = string(auth)
	return c.token, nil
}

func (c *Client) SetToken(token string) error {
	c.token = token
	if c.tokenPath == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(c.tokenPath), 0700); err != nil {
		return err
	}
	return os.WriteFile(c.tokenPath, []byte(token), 0600)
}

func (c *Client) removeToken() error {
	c.token = ""
	if c.tokenPath == "" {
		return nil
	}
	if err := os.Remove(c.tokenPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
