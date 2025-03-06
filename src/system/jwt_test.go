package system

import (
	"testing"
)

func TestGenerateJWT(t *testing.T) {
	key := []byte("secret")
	token, err := GenerateJWT(key)
	if err != nil {
		t.Error(err.Error())
	}
	t.Log(token)
}

func TestValidateJWT(t *testing.T) {
	key := []byte("secret")
	token, _ := GenerateJWT(key)
	if !ValidateJWT(token, key) {
		t.Error("токен не валиден")
	}
}
