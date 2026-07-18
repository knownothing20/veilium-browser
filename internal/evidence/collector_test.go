package evidence

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestCollectorServesControlledPagesAndAcceptsOneSubmission(t *testing.T) {
	collector, err := StartCollector(CollectorOptions{RequestedSurfaces: []string{"canvas", "canvas", "webgl"}})
	if err != nil {
		t.Fatal(err)
	}
	defer closeCollector(t, collector)
	if !strings.HasPrefix(collector.URL(), "http://127.0.0.1:") {
		t.Fatalf("collector is not loopback: %s", collector.URL())
	}

	client := &http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{Proxy: nil}}
	response, err := client.Get(collector.URL())
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected run page status %d", response.StatusCode)
	}
	if !strings.Contains(response.Header.Get("Content-Security-Policy"), "frame-ancestors 'none'") {
		t.Fatalf("main page CSP did not prevent embedding: %s", response.Header.Get("Content-Security-Policy"))
	}
	if !strings.Contains(string(body), "Veilium local evidence") || strings.Contains(string(body), "https://") {
		t.Fatalf("unexpected controlled page body")
	}

	frameResponse, err := client.Get(collector.origin + "/frame/" + collector.token)
	if err != nil {
		t.Fatal(err)
	}
	_ = frameResponse.Body.Close()
	if !strings.Contains(frameResponse.Header.Get("Content-Security-Policy"), "frame-ancestors 'self'") {
		t.Fatalf("frame page CSP did not restrict embedding to self: %s", frameResponse.Header.Get("Content-Security-Policy"))
	}
	workerResponse, err := client.Get(collector.origin + "/worker/" + collector.token + ".js")
	if err != nil {
		t.Fatal(err)
	}
	_ = workerResponse.Body.Close()
	if workerResponse.Header.Get("Content-Type") != "text/javascript; charset=utf-8" {
		t.Fatalf("unexpected worker content type %q", workerResponse.Header.Get("Content-Type"))
	}

	submission := validBrowserSubmission()
	status := postSubmission(t, client, collector, submission, collector.origin, collector.listener.Addr().String())
	if status != http.StatusNoContent {
		t.Fatalf("unexpected submission status %d", status)
	}
	waitContext, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result, err := collector.Wait(waitContext)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Contexts) != 1 || result.Contexts[0].Context != ContextTopLevel {
		t.Fatalf("unexpected submission result %#v", result)
	}
	if result.Contexts[0].Languages[0] != "en-US" {
		t.Fatalf("submission was not normalized: %#v", result.Contexts[0].Languages)
	}
	if status := postSubmission(t, client, collector, submission, collector.origin, collector.listener.Addr().String()); status != http.StatusGone {
		t.Fatalf("expected consumed token status, got %d", status)
	}
}

func TestCollectorRejectsHostAndOriginBeforeConsumption(t *testing.T) {
	collector, err := StartCollector(CollectorOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer closeCollector(t, collector)
	client := &http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{Proxy: nil}}
	submission := validBrowserSubmission()
	if status := postSubmission(t, client, collector, submission, collector.origin, "example.invalid"); status != http.StatusForbidden {
		t.Fatalf("expected host rejection, got %d", status)
	}
	if status := postSubmission(t, client, collector, submission, "http://example.invalid", collector.listener.Addr().String()); status != http.StatusForbidden {
		t.Fatalf("expected origin rejection, got %d", status)
	}
	if status := postSubmission(t, client, collector, submission, collector.origin, collector.listener.Addr().String()); status != http.StatusNoContent {
		t.Fatalf("valid submission was consumed by an invalid request: %d", status)
	}
}

func TestCollectorConsumesMalformedSubmissionAndReportsFailure(t *testing.T) {
	collector, err := StartCollector(CollectorOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer closeCollector(t, collector)
	request, err := http.NewRequest(http.MethodPost, collector.origin+"/submit/"+collector.token, strings.NewReader(`{"schemaVersion":1,"unknown":true}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", collector.origin)
	client := &http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{Proxy: nil}}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected malformed submission rejection, got %d", response.StatusCode)
	}
	waitContext, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := collector.Wait(waitContext); err == nil || !strings.Contains(err.Error(), "decode browser evidence submission") {
		t.Fatalf("expected collector decode failure, got %v", err)
	}
	if status := postSubmission(t, client, collector, validBrowserSubmission(), collector.origin, collector.listener.Addr().String()); status != http.StatusGone {
		t.Fatalf("malformed submission did not consume one-time token: %d", status)
	}
}

func TestCollectorWaitCancellationAndCloseAreBounded(t *testing.T) {
	collector, err := StartCollector(CollectorOptions{})
	if err != nil {
		t.Fatal(err)
	}
	waitContext, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := collector.Wait(waitContext); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancelled wait, got %v", err)
	}
	closeContext, closeCancel := context.WithTimeout(context.Background(), time.Second)
	defer closeCancel()
	if err := collector.Close(closeContext); err != nil {
		t.Fatal(err)
	}
	if err := collector.Close(closeContext); err != nil {
		t.Fatalf("collector close was not idempotent: %v", err)
	}
}

func TestCollectorRejectsUnknownRequestedSurface(t *testing.T) {
	if _, err := StartCollector(CollectorOptions{RequestedSurfaces: []string{"cookies"}}); err == nil || !strings.Contains(err.Error(), "unsupported requested surface") {
		t.Fatalf("expected requested-surface rejection, got %v", err)
	}
}

func postSubmission(t *testing.T, client *http.Client, collector *Collector, submission BrowserSubmission, origin, host string) int {
	t.Helper()
	payload, err := json.Marshal(submission)
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPost, collector.origin+"/submit/"+collector.token, bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", origin)
	request.Host = host
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)
	return response.StatusCode
}

func closeCollector(t *testing.T, collector *Collector) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := collector.Close(ctx); err != nil {
		t.Fatalf("close collector: %v", err)
	}
}

func validBrowserSubmission() BrowserSubmission {
	return BrowserSubmission{
		SchemaVersion: PayloadSchemaVersion,
		Contexts: []BrowserSnapshot{{
			Context:             ContextTopLevel,
			UserAgent:           "Mozilla/5.0 Chromium/148",
			UAPlatform:          "Windows",
			UABrands:            []string{"Chromium/148", "Chromium/148"},
			NavigatorPlatform:   "Win32",
			Language:            "en-US",
			Languages:           []string{" en-US ", "en-US"},
			Timezone:            "America/Los_Angeles",
			HardwareConcurrency: 8,
			Screen: &ScreenSnapshot{
				Width: 1920, Height: 1080, AvailWidth: 1920, AvailHeight: 1040, ColorDepth: 24, PixelDepth: 24,
			},
			Window: &WindowSnapshot{
				OuterWidth: 1280, OuterHeight: 900, InnerWidth: 1264, InnerHeight: 820,
				ViewportWidth: 1264, ViewportHeight: 820, ViewportScale: 1, DevicePixelRatio: 1,
			},
			WebRTC: &WebRTCSnapshot{Available: true, CandidateTypes: []string{"host"}, Protocols: []string{"udp"}, UsesMDNS: true, GatheringState: "complete"},
		}},
	}
}
