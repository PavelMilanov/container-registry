package client

import (
	"testing"
)

func TestGetCloudList(t *testing.T) {
	cr := NewClient()
	token, err := cr.Login("admin", "admin")
	if err != nil {
		t.Errorf("Login() error = %v", err)
	}
	data, err := cr.GetCloudList(token)
	if err != nil {
		t.Errorf("GetCloudList() error = %v", err)
	}
	t.Log(data)
}
