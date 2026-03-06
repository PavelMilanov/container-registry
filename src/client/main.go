package client

import (
	"context"
	"errors"
	"net/http"
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
