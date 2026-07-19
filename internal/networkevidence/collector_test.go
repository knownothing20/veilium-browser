package networkevidence

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestCollectorServesStrictControlledResources(t *testing.T) {
	collector, err := StartCollector(validProbeSet())
	if err != nil {
		t.Fatal(err)
	}
	defer collector.Close(context.Background())

	response, err := http.Get(collector.URL())
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected run-page status %d", response.StatusCode)
	}
	csp := response.Header.Get("Content-Security-Policy")
	for _, required := range []string{"default-src 'none'", "script-src 'self'", "frame-ancestors 'none'", "probe.example.invalid"} {
		if !strings.Contains(csp, required) {
			t.Fatalf("missing CSP restriction %q: %s", required, csp)
		}
	}
	if strings.Contains(string(body), "server.example") || strings.Contains(string(body), "credential") {
		t.Fatalf("run page exposed route material: %s", body)
	}

	token := strings.TrimPrefix(strings.TrimPrefix(collector.URL(), collector.origin+"/run/"), "/")
	configResponse, err := http.Get(collector.origin + "/config/" + token + ".json")
	if err != nil {
		t.Fatal(err)
	}
	var config browserProbeConfig
	if err := json.NewDecoder(configResponse.Body).Decode(&config); err != nil {
		t.Fatal(err)
	}
	_ = configResponse.Body.Close()
	if config.SchemaVersion != BrowserSubmissionSchemaVersion || len(config.Definitions) != 3 || len(config.DNSToken) != 32 {
		t.Fatalf("unexpected browser config: %#v", config)
	}
	if configResponse.Header.Get("Cache-Control") != "no-store, max-age=0" {
		t.Fatalf("unexpected cache policy: %q", configResponse.Header.Get("Cache-Control"))
	}
}

func TestCollectorRequiresHostOriginAndOneSubmission(t *testing.T) {
	collector, err := StartCollector(validProbeSet())
	if err != nil {
		t.Fatal(err)
	}
	defer collector.Close(context.Background())
	token := strings.TrimPrefix(collector.URL(), collector.origin+"/run/")
	submitURL := collector.origin + "/submit/" + token
	payload, err := json.Marshal(validBrowserSubmission())
	if err != nil {
		t.Fatal(err)
	}

	request, _ := http.NewRequest(http.MethodPost, submitURL, bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://invalid.example")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("expected invalid origin rejection, got %d", response.StatusCode)
	}

	request, _ = http.NewRequest(http.MethodPost, submitURL, bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", collector.origin)
	request.Host = "127.0.0.1:1"
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("expected invalid host rejection, got %d", response.StatusCode)
	}

	request, _ = http.NewRequest(http.MethodPost, submitURL, bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", collector.origin)
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("expected successful submission, got %d", response.StatusCode)
	}

	waitContext, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	submission, err := collector.Wait(waitContext)
	if err != nil || len(submission.Observations) != 3 {
		t.Fatalf("unexpected collector result: %#v, %v", submission, err)
	}

	request, _ = http.NewRequest(http.MethodPost, submitURL, bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", collector.origin)
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusGone {
		t.Fatalf("expected one-shot rejection, got %d", response.StatusCode)
	}
}

func TestCollectorConsumesInvalidSubmission(t *testing.T) {
	collector, err := StartCollector(validProbeSet())
	if err != nil {
		t.Fatal(err)
	}
	defer collector.Close(context.Background())
	token := strings.TrimPrefix(collector.URL(), collector.origin+"/run/")
	submitURL := collector.origin + "/submit/" + token
	request, _ := http.NewRequest(http.MethodPost, submitURL, strings.NewReader(`{"schemaVersion":1,"observations":[{"probeId":"unknown"}]}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", collector.origin)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected invalid submission rejection, got %d", response.StatusCode)
	}

	request, _ = http.NewRequest(http.MethodPost, submitURL, bytes.NewReader([]byte(`{}`)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", collector.origin)
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusGone {
		t.Fatalf("expected token consumption after invalid submission, got %d", response.StatusCode)
	}
}

func validBrowserSubmission() BrowserSubmission {
	return BrowserSubmission{
		SchemaVersion: BrowserSubmissionSchemaVersion,
		Observations: []BrowserObservation{
			{ProbeID: "exit", ProbeRevision: 1, ProbeKind: ProbeExitIP, Status: ObservationPassed, Values: []string{"203.0.113.8"}},
			{ProbeID: "stun", ProbeRevision: 1, ProbeKind: ProbeWebRTCSTUN, Status: ObservationPassed, Values: []string{"candidate:srflx", "protocol:udp", "public-ip:203.0.113.8"}},
			{ProbeID: "dns", ProbeRevision: 1, ProbeKind: ProbeDelegatedDNS, Status: ObservationPassed, Values: []string{"seen:true", "resolver-ip:192.0.2.53", "rcode:NOERROR"}},
		},
	}
}
