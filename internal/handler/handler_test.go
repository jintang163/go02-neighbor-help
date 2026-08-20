package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go02-neighbor-help/internal/auth"
	"go02-neighbor-help/internal/handler"
	"go02-neighbor-help/internal/server"
	"go02-neighbor-help/internal/service"
	"go02-neighbor-help/internal/store"
)

func TestHealthz(t *testing.T) {
	mem := store.NewMemoryStore(time.Now, nil)
	hasher := auth.NewPasswordHasher()
	sessions := auth.NewSessionManager(time.Hour)
	svc := service.NewServices(mem, hasher, sessions, nil, 10)
	h := handler.New(svc, mem, sessions, nil)
	mux := server.NewMux(h)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status %d", rec.Code)
	}
	var body map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["status"] != "ok" {
		t.Fatalf("body %s", rec.Body.String())
	}
}

func TestRegisterLoginMe(t *testing.T) {
	mem := store.NewMemoryStore(time.Now, nil)
	hasher := auth.NewPasswordHasher()
	sessions := auth.NewSessionManager(time.Hour)
	svc := service.NewServices(mem, hasher, sessions, nil, 10)
	h := handler.New(svc, mem, sessions, nil)
	mux := server.NewMux(h)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(`{"username":"erin","password":"erin1234","display_name":"Erin"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("register %d %s", rec.Code, rec.Body.String())
	}
	var authResp struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &authResp)
	if authResp.Token == "" {
		t.Fatal("empty token")
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req2.Header.Set("Authorization", "Bearer "+authResp.Token)
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != 200 {
		t.Fatalf("me %d %s", rec2.Code, rec2.Body.String())
	}
}
