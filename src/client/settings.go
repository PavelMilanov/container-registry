package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

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
	switch resp.StatusCode {
	case http.StatusAccepted:
		fmt.Println(string(body))
		return nil

	case http.StatusUnauthorized:
		c.removeToken()
		return errors.New("Необходима авторизация")
	default:
		return errors.New(string(body))
	}
}

func (c *Client) GetGarbageTagCount() error {
	auth, err := c.getToken()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/settings", c.ServerURL), nil)
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
	case http.StatusOK:
		fmt.Println(string(body))
		return nil

	case http.StatusUnauthorized:
		c.removeToken()
		return errors.New("Необходима авторизация")
	default:
		return errors.New(string(body))
	}
}
