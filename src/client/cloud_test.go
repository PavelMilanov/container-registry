package client

import (
	"testing"
)

func TestGetCloudList(t *testing.T) {
	token, err := Login("admin", "admin")
	if err != nil {
		t.Errorf("Login() error = %v", err)
	}
	data, err := GetCloudList(token)
	if err != nil {
		t.Errorf("GetCloudList() error = %v", err)
	}
	t.Log(data)
}
