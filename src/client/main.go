package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Client struct {
	ServerURL string
}

func NewClient() *Client {
	return &Client{ServerURL: "http://0.0.0.0:5050"}
}

func (c *Client) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.ServerURL+"/check", nil)
	if err != nil {
		return err
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("unhealthy")
	}
	return nil
}

func (c *Client) GarbageCollection() error {
	auth, err := c.getToken()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/api/garbage-collection", c.ServerURL), nil)
	if err != nil {
		return err
	}
	client := &http.Client{}
	req.Header.Add("Authorization", "Bearer "+auth)
	resp, err := client.Do(req)
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

func (c *Client) SetGarbageTagCount(tag string) error {
	auth, err := c.getToken()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/api/settings", c.ServerURL), nil)
	if err != nil {
		return err
	}
	client := &http.Client{}
	q := req.URL.Query()
	q.Add("tag", tag)
	req.URL.RawQuery = q.Encode()
	req.Header.Add("Authorization", "Bearer "+auth)
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
	fmt.Println(string(body))
	return nil
}

func (c *Client) getToken() (string, error) {
	auth, err := os.ReadFile("/tmp/.auth")
	if err != nil {
		return "", errors.New("Доступ запрещен. Необходимо авторизоваться.")
	}
	return string(auth), nil
}

func (c *Client) removeToken() error {
	if err := os.Remove("/tmp/.auth"); err != nil {
		return err
	}
	return nil
}
