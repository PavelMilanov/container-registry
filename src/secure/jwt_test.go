package secure

import (
	"testing"
)

func TestGenerateJWT(t *testing.T) {
	token, err := GenerateJWT("foo")
	if err != nil {
		t.Error(err.Error())
	}
	t.Log(token)
}

func TestValidateJWT(t *testing.T) {
	token, _ := GenerateJWT("foo")
	ValidateJWT(token)
}
