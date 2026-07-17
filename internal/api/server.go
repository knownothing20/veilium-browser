package api

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
	"github.com/knownothing20/veilium-browser/internal/launch"
	"github.com/knownothing20/veilium-browser/internal/profile"
	"github.com/knownothing20/veilium-browser/internal/security"
)

type Config struct {
	ListenAddr  string
	Token       string
	AllowRemote bool
	Store       *profile.Store
	Planner     launch.Planner
}

type Server struct {
	config Config
	http   *http.Server
}

func New(config Config) (*Server, error) {
	if config.Store == nil {
		return nil, fmt.Errorf("profile store is required")
	}
	if err := security.ValidateToken(config.Token); err != nil {
		return nil, err
	}
	if err := validateListenAddress(config.ListenAddr, config.AllowRemote); err != nil {
		return nil, err
	}

	server := &Server{config: config}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", server.health)
	mux.Handle("/v1/", server.authenticate(http.HandlerFunc(server.v1)))
	server.http = &http.Server{
		Addr:              config.ListenAddr,
		Handler:           limitBody(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return server, nil
}

func (s *Server) HTTPServer() *http.Server { return s.http }
func (s *Server) ListenAndServe() error    { return s.http.ListenAndServe() }

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) v1(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/v1/fingerprint/providers":
		providers := make([]fingerprint.Capabilities, 0, 2)
		for _, provider := range fingerprint.Providers() {
			version := "144.0.0"
			if provider == fingerprint.ProviderNative {
				version = "131.0.0"
			}
			capabilities, _ := fingerprint.For(provider, version)
			providers = append(providers, capabilities)
		}
		writeJSON(w, http.StatusOK, providers)
	case r.Method == http.MethodGet && r.URL.Path == "/v1/profiles":
		writeJSON(w, http.StatusOK, s.config.Store.List())
	case r.Method == http.MethodPost && r.URL.Path == "/v1/profiles":
		var input domain.Profile
		if err := decodeJSON(r, &input); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if _, err := fingerprint.Validate(withDefaultSeed(input)); err != nil {
			writeError(w, http.StatusUnprocessableEntity, err)
			return
		}
		created, err := s.config.Store.Create(input)
		if err != nil {
			writeError(w, http.StatusConflict, err)
			return
		}
		writeJSON(w, http.StatusCreated, created)
	case strings.HasPrefix(r.URL.Path, "/v1/profiles/"):
		s.handleProfile(w, r)
	default:
		writeError(w, http.StatusNotFound, errors.New("endpoint not found"))
	}
}

func (s *Server) handleProfile(w http.ResponseWriter, r *http.Request) {
	remainder := strings.TrimPrefix(r.URL.Path, "/v1/profiles/")
	parts := strings.Split(strings.Trim(remainder, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusNotFound, errors.New("profile not found"))
		return
	}
	id := parts[0]
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			item, err := s.config.Store.Get(id)
			if errors.Is(err, profile.ErrNotFound) {
				writeError(w, http.StatusNotFound, err)
				return
			}
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, item)
		case http.MethodDelete:
			if err := s.config.Store.Delete(id); errors.Is(err, profile.ErrNotFound) {
				writeError(w, http.StatusNotFound, err)
			} else if err != nil {
				writeError(w, http.StatusInternalServerError, err)
			} else {
				w.WriteHeader(http.StatusNoContent)
			}
		default:
			writeError(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
		}
		return
	}
	if len(parts) == 2 && parts[1] == "launch-plan" && r.Method == http.MethodPost {
		item, err := s.config.Store.Get(id)
		if errors.Is(err, profile.ErrNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		var request struct {
			RemoteDebuggingPort int `json:"remoteDebuggingPort"`
		}
		if r.ContentLength > 0 {
			if err := decodeJSON(r, &request); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
		}
		plan, err := s.config.Planner.Build(item, request.RemoteDebuggingPort)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, err)
			return
		}
		writeJSON(w, http.StatusOK, plan)
		return
	}
	writeError(w, http.StatusNotFound, errors.New("endpoint not found"))
}

func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		provided := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if subtle.ConstantTimeCompare([]byte(provided), []byte(s.config.Token)) != 1 {
			writeError(w, http.StatusUnauthorized, errors.New("invalid bearer token"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func validateListenAddress(address string, allowRemote bool) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("invalid listen address: %w", err)
	}
	if allowRemote {
		return nil
	}
	ip := net.ParseIP(host)
	if host == "localhost" || (ip != nil && ip.IsLoopback()) {
		return nil
	}
	return fmt.Errorf("refusing non-loopback listen address without AllowRemote")
}

func limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		next.ServeHTTP(w, r)
	})
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request must contain one JSON object")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func withDefaultSeed(item domain.Profile) domain.Profile {
	if item.Fingerprint.Seed == "" {
		item.Fingerprint.Seed = "profile-default"
	}
	return item
}
