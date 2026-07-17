package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/knownothing20/veilium-browser/internal/api"
	"github.com/knownothing20/veilium-browser/internal/launch"
	"github.com/knownothing20/veilium-browser/internal/profile"
	"github.com/knownothing20/veilium-browser/internal/security"
)

func main() {
	dataDir := getenv("VEILIUM_DATA_DIR", "./data")
	listenAddr := getenv("VEILIUM_LISTEN_ADDR", "127.0.0.1:51090")
	token := os.Getenv("VEILIUM_API_TOKEN")
	if token == "" {
		var err error
		token, err = security.GenerateToken()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(os.Stderr, "Veilium generated an ephemeral local API token: %s\n", token)
	}

	store, err := profile.Open(filepath.Join(dataDir, "profiles.json"))
	if err != nil {
		log.Fatal(err)
	}
	server, err := api.New(api.Config{
		ListenAddr: listenAddr,
		Token:      token,
		Store:      store,
		Planner:    launch.Planner{},
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Veilium local API listening on http://%s", listenAddr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
