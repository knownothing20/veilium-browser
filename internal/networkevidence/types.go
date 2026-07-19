package networkevidence

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"
)

const (
	SchemaVersion        = 1
	ProbeSchemaVersion   = 1
	MatrixSchemaVersion  = 1
	maxObservations      = 64
	maxObservationValues = 16
	maxLimitations       = 64
)

type RunStatus string

const (
	RunPending     RunStatus = "pending"
	RunRunning     RunStatus = "running"
	RunPassed      RunStatus = "passed"
	RunPartial     RunStatus = "partial"
	RunFailed      RunStatus = "failed"
	RunUnavailable RunStatus = "unavailable"
	RunCancelled   RunStatus = "cancelled"
	RunIncomplete  RunStatus = "incomplete"
)

type ObservationStatus string

const (
	ObservationPassed      ObservationStatus = "passed"
	ObservationPartial     ObservationStatus = "partial"
	ObservationFailed      ObservationStatus = "failed"
	ObservationUnavailable ObservationStatus = "unavailable"
	ObservationSkipped     ObservationStatus = "skipped"
)

type ProbeKind string

const (
	ProbeExitIP       ProbeKind = "exit-ip"
	ProbeWebRTCSTUN   ProbeKind = "webrtc-stun"
	ProbeDelegatedDNS ProbeKind = "delegated-dns"
)

type RouteKind string

const (
	RouteDirect          RouteKind = "direct"
	RouteHTTP            RouteKind = "http-proxy"
	RouteHTTPS           RouteKind = "https-proxy"
	RouteSOCKS5          RouteKind = "socks5-proxy"
	RouteLocalAuthBridge RouteKind = "local-auth-bridge"
	RouteXray            RouteKind = "xray"
	RouteSingBox         RouteKind = "sing-box"
)

type RouteIdentity struct {
	Kind       RouteKind `json:"kind"`
	Scheme     string    `json:"scheme"`
	BridgeKind string    `json:"bridgeKind,omitempty"`
	Digest     string    `json:"digest"`
}

type Observation struct {
	ID           string            `json:"id"`
	ProbeKind    ProbeKind         `json:"probeKind"`
	ProbeID      string            `json:"probeId"`
	ProbeRevision int              `json:"probeRevision"`
	Status       ObservationStatus `json:"status"`
	Expected     string            `json:"expected,omitempty"`
	Values       []string          `json:"values,omitempty"`
	ReasonCode   string            `json:"reasonCode,omitempty"`
	Detail       string            `json:"detail,omitempty"`
	CollectedAt  time.Time         `json:"collectedAt"`
}

type Run struct {
	SchemaVersion          int           `json:"schemaVersion"`
	ID                     string        `json:"id"`
	EvidenceRunID          string        `json:"evidenceRunId"`
	ProfileID              string        `json:"profileId"`
	ProviderID             string        `json:"providerId"`
	ProviderRevision       int           `json:"providerRevision"`
	BrowserVersion         string        `json:"browserVersion"`
	OperatingSystem        string        `json:"operatingSystem"`
	Architecture           string        `json:"architecture"`
	BinaryIdentityDigest   string        `json:"binaryIdentityDigest"`
	ConsistencyInputDigest string        `json:"consistencyInputDigest"`
	Route                  RouteIdentity `json:"route"`
	ProbeSetID             string        `json:"probeSetId"`
	ProbeSetRevision       int           `json:"probeSetRevision"`
	Status                 RunStatus     `json:"status"`
	StartedAt              time.Time     `json:"startedAt"`
	CompletedAt            *time.Time    `json:"completedAt,omitempty"`
	ExpiresAt              time.Time     `json:"expiresAt"`
	Observations           []Observation `json:"observations"`
	Limitations            []string      `json:"limitations,omitempty"`
	FailureCode            string        `json:"failureCode,omitempty"`
	FailureDetail          string        `json:"failureDetail,omitempty"`
}

func NewRunID() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate network evidence id: %w", err)
	}
	return "netev-" + hex.EncodeToString(buffer), nil
}

func (run Run) Validate() error {
	if run.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported network evidence schema %d", run.SchemaVersion)
	}
	for label, value := range map[string]string{
		"run id": run.ID, "evidence run id": run.EvidenceRunID, "profile id": run.ProfileID,
		"provider id": run.ProviderID, "browser version": run.BrowserVersion,
		"operating system": run.OperatingSystem, "architecture": run.Architecture,
		"probe set id": run.ProbeSetID,
	} {
		if strings.TrimSpace(value) == "" || len(value) > 256 {
			return fmt.Errorf("network evidence %s is invalid", label)
		}
	}
	if run.ProviderRevision < 1 || run.ProbeSetRevision < 1 {
		return fmt.Errorf("network evidence revisions are required")
	}
	for label, digest := range map[string]string{
		"binary identity": run.BinaryIdentityDigest,
		"consistency input": run.ConsistencyInputDigest,
	} {
		if !validSHA256(digest) {
			return fmt.Errorf("network evidence %s digest is invalid", label)
		}
	}
	if err := run.Route.Validate(); err != nil {
		return err
	}
	if !validRunStatus(run.Status) {
		return fmt.Errorf("network evidence status %q is invalid", run.Status)
	}
	if run.StartedAt.IsZero() || run.ExpiresAt.IsZero() || !run.ExpiresAt.After(run.StartedAt) {
		return fmt.Errorf("network evidence timestamps are invalid")
	}
	if terminalRunStatus(run.Status) && run.CompletedAt == nil {
		return fmt.Errorf("terminal network evidence requires completion time")
	}
	if run.Status == RunFailed && strings.TrimSpace(run.FailureCode) == "" {
		return fmt.Errorf("failed network evidence requires a failure code")
	}
	if len(run.Observations) > maxObservations {
		return fmt.Errorf("network evidence has too many observations")
	}
	seen := make(map[string]struct{}, len(run.Observations))
	for index, observation := range run.Observations {
		if _, exists := seen[observation.ID]; exists {
			return fmt.Errorf("duplicate network observation %q", observation.ID)
		}
		seen[observation.ID] = struct{}{}
		if err := observation.Validate(); err != nil {
			return fmt.Errorf("network observation %d: %w", index, err)
		}
	}
	if len(run.Limitations) > maxLimitations {
		return fmt.Errorf("network evidence has too many limitations")
	}
	for _, limitation := range run.Limitations {
		if len(strings.TrimSpace(limitation)) > 512 {
			return fmt.Errorf("network evidence limitation is too long")
		}
	}
	if len(run.FailureCode) > 128 || len(run.FailureDetail) > 1024 {
		return fmt.Errorf("network evidence failure detail is too long")
	}
	return nil
}

func (route RouteIdentity) Validate() error {
	if !validRouteKind(route.Kind) {
		return fmt.Errorf("network evidence route kind %q is invalid", route.Kind)
	}
	if strings.TrimSpace(route.Scheme) == "" || len(route.Scheme) > 32 {
		return fmt.Errorf("network evidence route scheme is invalid")
	}
	if len(route.BridgeKind) > 64 || !validSHA256(route.Digest) {
		return fmt.Errorf("network evidence route identity is invalid")
	}
	return nil
}

func (observation Observation) Validate() error {
	if strings.TrimSpace(observation.ID) == "" || len(observation.ID) > 128 {
		return fmt.Errorf("network observation id is invalid")
	}
	if !validProbeKind(observation.ProbeKind) || strings.TrimSpace(observation.ProbeID) == "" || observation.ProbeRevision < 1 {
		return fmt.Errorf("network observation probe identity is invalid")
	}
	if !validObservationStatus(observation.Status) {
		return fmt.Errorf("network observation status %q is invalid", observation.Status)
	}
	if observation.CollectedAt.IsZero() {
		return fmt.Errorf("network observation collection time is required")
	}
	if len(observation.Values) > maxObservationValues {
		return fmt.Errorf("network observation has too many values")
	}
	for _, value := range observation.Values {
		if len(strings.TrimSpace(value)) > 256 {
			return fmt.Errorf("network observation value is too long")
		}
	}
	if len(observation.Expected) > 512 || len(observation.ReasonCode) > 128 || len(observation.Detail) > 1024 {
		return fmt.Errorf("network observation metadata is too long")
	}
	return validateObservationValues(observation)
}

func validateObservationValues(observation Observation) error {
	switch observation.ProbeKind {
	case ProbeExitIP:
		if observation.Status == ObservationPassed || observation.Status == ObservationPartial || observation.Status == ObservationFailed {
			if len(observation.Values) != 1 || net.ParseIP(observation.Values[0]) == nil {
				return fmt.Errorf("exit-IP observation requires one normalized IP address")
			}
		}
	case ProbeWebRTCSTUN:
		for _, value := range observation.Values {
			if validWebRTCValue(value) {
				continue
			}
			return fmt.Errorf("invalid WebRTC/STUN observation value")
		}
	case ProbeDelegatedDNS:
		for _, value := range observation.Values {
			if validDNSValue(value) {
				continue
			}
			return fmt.Errorf("invalid delegated-DNS observation value")
		}
	}
	return nil
}

func Normalize(run Run) Run {
	run.Limitations = sortedUnique(run.Limitations)
	for index := range run.Observations {
		run.Observations[index].Values = sortedUnique(run.Observations[index].Values)
	}
	sort.Slice(run.Observations, func(i, j int) bool { return run.Observations[i].ID < run.Observations[j].ID })
	return run
}

func validWebRTCValue(value string) bool {
	value = strings.TrimSpace(value)
	switch value {
	case "candidate:host", "candidate:srflx", "candidate:prflx", "candidate:relay",
		"protocol:udp", "protocol:tcp", "mdns:true", "mdns:false":
		return true
	}
	if strings.HasPrefix(value, "public-ip:") {
		return net.ParseIP(strings.TrimPrefix(value, "public-ip:")) != nil
	}
	return false
}

func validDNSValue(value string) bool {
	value = strings.TrimSpace(value)
	if value == "seen:true" || value == "seen:false" {
		return true
	}
	if strings.HasPrefix(value, "resolver-ip:") {
		return net.ParseIP(strings.TrimPrefix(value, "resolver-ip:")) != nil
	}
	if strings.HasPrefix(value, "rcode:") {
		rcode := strings.TrimPrefix(value, "rcode:")
		return rcode != "" && len(rcode) <= 32 && strings.ToUpper(rcode) == rcode
	}
	return false
}

func validSHA256(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validRunStatus(value RunStatus) bool {
	switch value {
	case RunPending, RunRunning, RunPassed, RunPartial, RunFailed, RunUnavailable, RunCancelled, RunIncomplete:
		return true
	default:
		return false
	}
}

func terminalRunStatus(value RunStatus) bool { return value != RunPending && value != RunRunning }

func validObservationStatus(value ObservationStatus) bool {
	switch value {
	case ObservationPassed, ObservationPartial, ObservationFailed, ObservationUnavailable, ObservationSkipped:
		return true
	default:
		return false
	}
}

func validProbeKind(value ProbeKind) bool {
	switch value {
	case ProbeExitIP, ProbeWebRTCSTUN, ProbeDelegatedDNS:
		return true
	default:
		return false
	}
}

func validRouteKind(value RouteKind) bool {
	switch value {
	case RouteDirect, RouteHTTP, RouteHTTPS, RouteSOCKS5, RouteLocalAuthBridge, RouteXray, RouteSingBox:
		return true
	default:
		return false
	}
}

func sortedUnique(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
