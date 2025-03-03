package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/PavelMilanov/container-registry/db"
	"github.com/PavelMilanov/container-registry/handlers"
	"github.com/PavelMilanov/container-registry/storage"
)

var env = config.NewEnv(config.DATA_PATH)
var s = storage.NewStorage(env)
var sqlite = db.NewDatabase("test.db")
var h = handlers.NewHandler(s, &sqlite, env)

func TestRegistration(t *testing.T) {
	srv := h.InitRouters()
	token := ""
	t.Run("registration", func(t *testing.T) {
		w := httptest.NewRecorder()
		body := `{"username": "test","password":"test","confirmPassword":"test"}`
		req, _ := http.NewRequest("POST", "/registration", strings.NewReader(body))
		srv.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatal("ошибка при регистрации")
		}
	})
	t.Run("login", func(t *testing.T) {
		w := httptest.NewRecorder()
		body := `{"username": "test","password":"test"}`
		req, _ := http.NewRequest("POST", "/login", strings.NewReader(body))
		srv.ServeHTTP(w, req)
		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Fatal("не указан логин или пароль")
		}
		if w.Code != http.StatusOK {
			t.Fatal("ошибка при входе")
		}
		token = response["token"].(string)
		t.Log(token)
	})
	os.Remove("test.db")
}
