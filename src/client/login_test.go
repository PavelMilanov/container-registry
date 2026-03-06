package client

import (
	"testing"
)

func TestLogin(t *testing.T) {
	cr := NewClient()
	t.Run("Failure", func(t *testing.T) {
		_, err := cr.Login("test", "test")
		if err != nil {
			t.Logf("Error: %s", err.Error())
		}
	})
	t.Run("Success", func(t *testing.T) {
		token, err := cr.Login("admin", "admin")
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Token: %s", token)
	})
}
