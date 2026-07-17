package supervisor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const maxVersionResponseBytes = 64 << 10

type HTTPProber struct {
	Client   *http.Client
	Interval time.Duration
}

func (p HTTPProber) Wait(ctx context.Context, port int) (VersionInfo, error) {
	if p.Client == nil {
		return VersionInfo{}, fmt.Errorf("CDP HTTP client is required")
	}
	interval := p.Interval
	if interval <= 0 {
		interval = 150 * time.Millisecond
	}
	endpoint := fmt.Sprintf("http://127.0.0.1:%d/json/version", port)
	var lastErr error
	for {
		version, err := p.probe(ctx, endpoint)
		if err == nil {
			return version, nil
		}
		lastErr = err
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			if lastErr != nil {
				return VersionInfo{}, fmt.Errorf("%v: %w", lastErr, ctx.Err())
			}
			return VersionInfo{}, ctx.Err()
		case <-timer.C:
		}
	}
}

func (p HTTPProber) probe(ctx context.Context, endpoint string) (VersionInfo, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return VersionInfo{}, err
	}
	request.Header.Set("Accept", "application/json")
	response, err := p.Client.Do(request)
	if err != nil {
		return VersionInfo{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return VersionInfo{}, fmt.Errorf("CDP readiness endpoint returned %s", response.Status)
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, maxVersionResponseBytes))
	var version VersionInfo
	if err := decoder.Decode(&version); err != nil {
		return VersionInfo{}, fmt.Errorf("decode CDP readiness response: %w", err)
	}
	return version, nil
}
