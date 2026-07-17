package api

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/launch"
	"github.com/knownothing20/veilium-browser/internal/profile"
)

func TestAPIRequiresBearerToken(t *testing.T) {
	store, err := profile.Open(filepath.Join(t.TempDir(), "profiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	server, err := New(Config{
		ListenAddr: "127.0.0.1:51090",
		Token:      "012345678901234567890123456789",
		Store:      store,
		Planner:    launch.Planner{},
	})
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/profiles", nil)
	server.HTTPServer().Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestRejectsRemoteListenByDefault(t *testing.T) {
	store, err := profile.Open(filepath.Join(t.TempDir(), "profiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = New(Config{
		ListenAddr: "0.0.0.0:51090",
		Token:      "012345678901234567890123456789",
		Store:      store,
	})
	if err == nil {
		t.Fatal("expected remote listen rejection")
	}
}
