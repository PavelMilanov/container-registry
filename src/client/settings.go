package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
)

func (c *Client) SetGarbageTagCount(ctx context.Context, tag string) error {
	req, cancel, err := c.newAuthorizedRequest(ctx, http.MethodPost, "/api/settings", nil)
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

func (c *Client) GetGarbageTagCount(ctx context.Context) error {
	req, cancel, err := c.newAuthorizedRequest(ctx, http.MethodGet, "/api/settings", nil)
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
