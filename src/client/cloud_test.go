package client

import (
	"testing"
)

func TestGetCloudList(t *testing.T) {
	cr := NewClient()
	data, err := cr.GetCloudList()
	if err != nil {
		t.Errorf("GetCloudList() error = %v", err)
	}
	t.Log(data)
}

func TestGetImagesList(t *testing.T) {
	cr := NewClient()
	data, err := cr.GetImagesList("dev", "registry")
	if err != nil {
		t.Errorf("GetImagesList() error = %v", err)
	}
	t.Log(data)
}

func TestDelImage(t *testing.T) {
	cr := NewClient()
	if err := cr.DelImage("dev", "registry", "latest"); err != nil {
		t.Errorf("DelImage() error = %v", err)
	}
}
