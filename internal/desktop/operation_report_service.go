package desktop

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/knownothing20/veilium-browser/internal/lifecycle"
)

const (
	operationReportSchemaVersion = 1
	operationReportKind          = "veilium-operation-report"
	maxOperationReportBytes      = 1 << 20
)

type OperationReportExportRequest struct {
	OperationID     string `json:"operationId"`
	DestinationPath string `json:"destinationPath"`
}

type OperationReportSummary struct {
	ItemCount        int   `json:"itemCount"`
	Succeeded        int   `json:"succeeded"`
	Skipped          int   `json:"skipped"`
	Cancelled        int   `json:"cancelled"`
	Failed           int   `json:"failed"`
	RolledBack       int   `json:"rolledBack"`
	RecoveryRequired int   `json:"recoveryRequired"`
	FilesProcessed   int64 `json:"filesProcessed"`
	BytesProcessed   int64 `json:"bytesProcessed"`
}

type OperationReportExportResult struct {
	OperationID   string                    `json:"operationId"`
	Path          string                    `json:"path"`
	PayloadSHA256 string                    `json:"payloadSha256"`
	GeneratedAt   time.Time                 `json:"generatedAt"`
	Status        lifecycle.OperationStatus `json:"status"`
}

type lifecycleOperationReportPayload struct {
	Operation lifecycle.Operation    `json:"operation"`
	Summary   OperationReportSummary `json:"summary"`
}

type lifecycleOperationReportArtifact struct {
	SchemaVersion int                             `json:"schemaVersion"`
	Kind          string                          `json:"kind"`
	GeneratedAt   time.Time                       `json:"generatedAt"`
	Payload       lifecycleOperationReportPayload `json:"payload"`
	PayloadSHA256 string                          `json:"payloadSha256"`
	Exclusions    []string                        `json:"exclusions"`
	Limitations   []string                        `json:"limitations"`
}

func (s *Service) ExportLifecycleOperationReport(request OperationReportExportRequest) (OperationReportExportResult, error) {
	if s.lifecycleJournal == nil {
		return OperationReportExportResult{}, fmt.Errorf("lifecycle operation journal is unavailable")
	}
	operationID := strings.TrimSpace(request.OperationID)
	if operationID == "" {
		return OperationReportExportResult{}, fmt.Errorf("lifecycle operation ID is required")
	}
	operation, err := s.lifecycleJournal.Get(operationID)
	if err != nil {
		return OperationReportExportResult{}, fmt.Errorf("read lifecycle operation %q: %w", operationID, err)
	}
	artifact, err := buildLifecycleOperationReport(operation, time.Now().UTC())
	if err != nil {
		return OperationReportExportResult{}, err
	}
	data, err := encodeLifecycleOperationReport(artifact)
	if err != nil {
		return OperationReportExportResult{}, err
	}
	path, err := writeLifecycleOperationReport(request.DestinationPath, data)
	if err != nil {
		return OperationReportExportResult{}, err
	}
	return OperationReportExportResult{
		OperationID:   operation.ID,
		Path:          path,
		PayloadSHA256: artifact.PayloadSHA256,
		GeneratedAt:   artifact.GeneratedAt,
		Status:        operation.Status,
	}, nil
}

func buildLifecycleOperationReport(operation lifecycle.Operation, generatedAt time.Time) (lifecycleOperationReportArtifact, error) {
	if err := operation.Validate(); err != nil {
		return lifecycleOperationReportArtifact{}, fmt.Errorf("validate lifecycle operation report source: %w", err)
	}
	operation.IdempotencyKey = ""
	operation.StagingRef = ""
	operation.QuarantineRef = ""
	operation.ProfileIDs = append([]string(nil), operation.ProfileIDs...)
	operation.Limitations = append([]string(nil), operation.Limitations...)
	operation.RecoveryActions = append([]string(nil), operation.RecoveryActions...)
	operation.Items = append([]lifecycle.OperationItemResult(nil), operation.Items...)
	for index := range operation.Items {
		operation.Items[index].Limitations = append([]string(nil), operation.Items[index].Limitations...)
		operation.Items[index].OutputID = operationReportIdentity(operation.Items[index].OutputID)
		operation.Items[index].RecoveryID = operationReportIdentity(operation.Items[index].RecoveryID)
	}
	if err := operation.Validate(); err != nil {
		return lifecycleOperationReportArtifact{}, fmt.Errorf("validate redacted lifecycle operation report: %w", err)
	}
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	payload := lifecycleOperationReportPayload{
		Operation: operation,
		Summary:   summarizeLifecycleOperation(operation),
	}
	digest, err := lifecycleOperationReportPayloadDigest(payload)
	if err != nil {
		return lifecycleOperationReportArtifact{}, err
	}
	return lifecycleOperationReportArtifact{
		SchemaVersion: operationReportSchemaVersion,
		Kind:          operationReportKind,
		GeneratedAt:   generatedAt.UTC(),
		Payload:       payload,
		PayloadSHA256: digest,
		Exclusions: []string{
			"browser-user-data-and-page-content",
			"credential-values-and-proxy-secrets",
			"operation-idempotency-key",
			"local-staging-and-quarantine-references",
			"runtime-logs-and-evidence-payloads",
			"absolute-local-output-and-recovery-paths",
		},
		Limitations: []string{
			"This artifact is a point-in-time local operation summary, not a portable Profile or backup.",
			"The report does not certify Profile health, Provider trust, compatibility, or Evidence validity.",
		},
	}, nil
}

func summarizeLifecycleOperation(operation lifecycle.Operation) OperationReportSummary {
	summary := OperationReportSummary{ItemCount: len(operation.Items)}
	for _, item := range operation.Items {
		summary.FilesProcessed += item.FilesProcessed
		summary.BytesProcessed += item.BytesProcessed
		switch item.Status {
		case lifecycle.ItemSucceeded:
			summary.Succeeded++
		case lifecycle.ItemSkipped:
			summary.Skipped++
		case lifecycle.ItemCancelled:
			summary.Cancelled++
		case lifecycle.ItemFailed:
			summary.Failed++
		case lifecycle.ItemRolledBack:
			summary.RolledBack++
		case lifecycle.ItemRecoveryRequired:
			summary.RecoveryRequired++
		}
	}
	return summary
}

func operationReportIdentity(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if filepath.IsAbs(value) || filepath.VolumeName(value) != "" || strings.ContainsAny(value, `/\\`) {
		return "redacted-local-reference"
	}
	return value
}

func lifecycleOperationReportPayloadDigest(payload lifecycleOperationReportPayload) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode lifecycle operation report payload: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func encodeLifecycleOperationReport(artifact lifecycleOperationReportArtifact) ([]byte, error) {
	if err := validateLifecycleOperationReport(artifact); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode lifecycle operation report: %w", err)
	}
	data = append(data, '\n')
	if len(data) > maxOperationReportBytes {
		return nil, fmt.Errorf("lifecycle operation report exceeds %d bytes", maxOperationReportBytes)
	}
	return data, nil
}

func decodeLifecycleOperationReport(data []byte) (lifecycleOperationReportArtifact, error) {
	if len(data) == 0 || len(data) > maxOperationReportBytes {
		return lifecycleOperationReportArtifact{}, fmt.Errorf("lifecycle operation report size is invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var artifact lifecycleOperationReportArtifact
	if err := decoder.Decode(&artifact); err != nil {
		return lifecycleOperationReportArtifact{}, fmt.Errorf("decode lifecycle operation report: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return lifecycleOperationReportArtifact{}, fmt.Errorf("lifecycle operation report contains trailing JSON")
		}
		return lifecycleOperationReportArtifact{}, fmt.Errorf("decode lifecycle operation report trailing data: %w", err)
	}
	if err := validateLifecycleOperationReport(artifact); err != nil {
		return lifecycleOperationReportArtifact{}, err
	}
	return artifact, nil
}

func validateLifecycleOperationReport(artifact lifecycleOperationReportArtifact) error {
	if artifact.SchemaVersion != operationReportSchemaVersion {
		return fmt.Errorf("unsupported lifecycle operation report schema %d", artifact.SchemaVersion)
	}
	if artifact.Kind != operationReportKind {
		return fmt.Errorf("unexpected lifecycle operation report kind %q", artifact.Kind)
	}
	if artifact.GeneratedAt.IsZero() {
		return fmt.Errorf("lifecycle operation report generation timestamp is required")
	}
	if err := artifact.Payload.Operation.Validate(); err != nil {
		return fmt.Errorf("validate lifecycle operation report payload: %w", err)
	}
	if artifact.Payload.Summary != summarizeLifecycleOperation(artifact.Payload.Operation) {
		return fmt.Errorf("lifecycle operation report summary does not match item results")
	}
	digest, err := lifecycleOperationReportPayloadDigest(artifact.Payload)
	if err != nil {
		return err
	}
	if !strings.EqualFold(strings.TrimSpace(artifact.PayloadSHA256), digest) {
		return fmt.Errorf("lifecycle operation report payload digest mismatch")
	}
	if len(artifact.Exclusions) == 0 || len(artifact.Exclusions) > 32 {
		return fmt.Errorf("lifecycle operation report exclusions are invalid")
	}
	if len(artifact.Limitations) == 0 || len(artifact.Limitations) > 32 {
		return fmt.Errorf("lifecycle operation report limitations are invalid")
	}
	return nil
}

func writeLifecycleOperationReport(path string, data []byte) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("lifecycle operation report destination is required")
	}
	if !strings.HasSuffix(strings.ToLower(path), ".json") {
		path += ".json"
	}
	absolute, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("resolve lifecycle operation report destination: %w", err)
	}
	if info, statErr := os.Lstat(absolute); statErr == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return "", fmt.Errorf("lifecycle operation report destination must be a regular file")
		}
		return "", fmt.Errorf("lifecycle operation report destination already exists")
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return "", fmt.Errorf("inspect lifecycle operation report destination: %w", statErr)
	}
	directory := filepath.Dir(absolute)
	info, err := os.Lstat(directory)
	if err != nil {
		return "", fmt.Errorf("inspect lifecycle operation report directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", fmt.Errorf("lifecycle operation report directory must be a real directory")
	}
	resolved, err := filepath.EvalSymlinks(directory)
	if err != nil {
		return "", fmt.Errorf("resolve lifecycle operation report directory: %w", err)
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("resolve lifecycle operation report directory: %w", err)
	}
	if !sameOperationReportPath(directory, resolved) {
		return "", fmt.Errorf("lifecycle operation report directory cannot pass through a link or reparse alias")
	}
	temp, err := os.CreateTemp(directory, ".veilium-operation-report-*.tmp")
	if err != nil {
		return "", fmt.Errorf("create lifecycle operation report staging file: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return "", fmt.Errorf("secure lifecycle operation report staging file: %w", err)
	}
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return "", fmt.Errorf("write lifecycle operation report staging file: %w", err)
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return "", fmt.Errorf("sync lifecycle operation report staging file: %w", err)
	}
	if err := temp.Close(); err != nil {
		return "", fmt.Errorf("close lifecycle operation report staging file: %w", err)
	}
	if runtime.GOOS == "windows" {
		if err := os.Rename(tempPath, absolute); err != nil {
			return "", fmt.Errorf("publish lifecycle operation report: %w", err)
		}
	} else {
		if err := os.Link(tempPath, absolute); err != nil {
			return "", fmt.Errorf("publish lifecycle operation report without overwrite: %w", err)
		}
	}
	return absolute, nil
}

func sameOperationReportPath(left, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}
