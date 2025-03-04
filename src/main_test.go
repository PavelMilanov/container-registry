package main

import (
	"encoding/json"
	"fmt"
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

func TestRegistrationAPI(t *testing.T) {
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

func TestRegistyAPI(t *testing.T) {
	srv := h.InitRouters()
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NDExNjA4NTUsImlhdCI6MTc0MTA3NDQ1NX0.IolFwVdG4GxiSMqlqlpD3YQlxmKFSlkipFZsh3GJMmM"
	t.Run("add registry", func(t *testing.T) {
		t.Run("with authorization", func(t *testing.T) {
			t.Parallel()
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/registry/test", nil)
			jwt := fmt.Sprintf("Bearer %s", token)
			req.Header.Set("Authorization", jwt)
			srv.ServeHTTP(w, req)

			var response db.Registry
			err := json.Unmarshal(w.Body.Bytes(), &response)
			if err != nil {
				t.Error(err)
			}
			if w.Code != http.StatusCreated {
				t.Errorf("Ошибка при создании репозитория: %s", w.Body.Bytes())
			}
			t.Logf("%+v", response)
		})
		t.Run("without authorization", func(t *testing.T) {
			t.Parallel()
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/registry/test", nil)
			srv.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Error("Неавторизованный запрос")
			}
		})
	})
	t.Run("get registry", func(t *testing.T) {
		t.Run("with authorization", func(t *testing.T) {
			t.Parallel()
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/registry/test", nil)
			jwt := fmt.Sprintf("Bearer %s", token)
			req.Header.Set("Authorization", jwt)
			srv.ServeHTTP(w, req)

			response := string(w.Body.Bytes())
			if w.Code != http.StatusOK {
				t.Errorf("Ошибка при получения репозиториев: %s", response)
			}
			t.Logf("%s", response)
		})
		t.Run("without authorization", func(t *testing.T) {
			t.Parallel()
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/registry/test", nil)
			srv.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Error("Неавторизованный запрос")
			}
		})
	})
	t.Run("delete registry", func(t *testing.T) {
		t.Run("with authorization", func(t *testing.T) {
			t.Parallel()
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("DELETE", "/api/registry/test", nil)
			jwt := fmt.Sprintf("Bearer %s", token)
			req.Header.Set("Authorization", jwt)
			srv.ServeHTTP(w, req)

			var response db.Registry
			err := json.Unmarshal(w.Body.Bytes(), &response)
			if err != nil {
				t.Error(err)
			}
			if w.Code != http.StatusAccepted {
				t.Errorf("Ошибка при удалении репозитория: %+v", response)
			}
			t.Logf("%+v", response)
		})
		t.Run("without authorization", func(t *testing.T) {
			t.Parallel()
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("DELETE", "/api/registry/test", nil)
			srv.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Error("Неавторизованный запрос")
			}
		})
	})
	os.Remove("test.db")
}
