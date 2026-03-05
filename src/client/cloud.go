package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"
)

func GetCloudList(auth string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:5050/api/cloud/list", nil)
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
		Data []string `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}
