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

func TestGetImagesList(t *testing.T) {
	cr := NewClient()
	token, err := cr.Login("admin", "admin")
	if err != nil {
		t.Errorf("Login() error = %v", err)
	}
	data, err := cr.GetImagesList(token, "dev", "registry")
	if err != nil {
		t.Errorf("GetImagesList() error = %v", err)
	}
	t.Log(data)
}

func TestDelImage(t *testing.T) {
	cr := NewClient()
	token, err := cr.Login("admin", "admin")
	if err != nil {
		t.Errorf("Login() error = %v", err)
	}
	err = cr.DelImage(token, "dev", "registry", "latest")
	if err != nil {
		t.Errorf("DelImage() error = %v", err)
	}
}
