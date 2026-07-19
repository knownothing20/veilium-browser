package networkevidence

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	probeIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)
	dnsNamePattern = regexp.MustCompile(`^(?i:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)+)$`)
)

type ProbeDefinition struct {
	SchemaVersion    int       `json:"schemaVersion"`
	ID               string    `json:"id"`
	Revision         int       `json:"revision"`
	Kind             ProbeKind `json:"kind"`
	HTTPSURL         string    `json:"httpsUrl,omitempty"`
	STUNServer       string    `json:"stunServer,omitempty"`
	DNSZone          string    `json:"dnsZone,omitempty"`
	DNSResultURL     string    `json:"dnsResultUrl,omitempty"`
	TimeoutSeconds   int       `json:"timeoutSeconds"`
	MaxResponseBytes int       `json:"maxResponseBytes,omitempty"`
	SelfHostable     bool      `json:"selfHostable"`
	PrivacyNote      string    `json:"privacyNote"`
}

type ProbeSet struct {
	SchemaVersion int               `json:"schemaVersion"`
	ID            string            `json:"id"`
	Revision      int               `json:"revision"`
	Definitions   []ProbeDefinition `json:"definitions"`
}

func (definition ProbeDefinition) Validate() error {
	if definition.SchemaVersion != ProbeSchemaVersion {
		return fmt.Errorf("unsupported probe schema %d", definition.SchemaVersion)
	}
	if !probeIDPattern.MatchString(definition.ID) || definition.Revision < 1 {
		return fmt.Errorf("probe identity is invalid")
	}
	if !validProbeKind(definition.Kind) {
		return fmt.Errorf("probe kind %q is invalid", definition.Kind)
	}
	if definition.TimeoutSeconds < 1 || definition.TimeoutSeconds > 60 {
		return fmt.Errorf("probe timeout must be between one and sixty seconds")
	}
	if len(strings.TrimSpace(definition.PrivacyNote)) < 8 || len(definition.PrivacyNote) > 512 {
		return fmt.Errorf("probe privacy note is required and bounded")
	}
	if !definition.SelfHostable {
		return fmt.Errorf("M4.4 probe definitions must be replaceable or self-hostable")
	}

	httpsURL := strings.TrimSpace(definition.HTTPSURL)
	stunServer := strings.TrimSpace(definition.STUNServer)
	dnsZone := strings.TrimSpace(definition.DNSZone)
	dnsResultURL := strings.TrimSpace(definition.DNSResultURL)
	switch definition.Kind {
	case ProbeExitIP:
		if stunServer != "" || dnsZone != "" || dnsResultURL != "" {
			return fmt.Errorf("exit-IP probe contains unrelated endpoint fields")
		}
		if err := validateProbeURL(httpsURL, "exit-IP"); err != nil {
			return err
		}
		if definition.MaxResponseBytes < 64 || definition.MaxResponseBytes > 64<<10 {
			return fmt.Errorf("exit-IP response limit is invalid")
		}
	case ProbeWebRTCSTUN:
		if httpsURL != "" || dnsZone != "" || dnsResultURL != "" || definition.MaxResponseBytes != 0 {
			return fmt.Errorf("STUN probe contains unrelated endpoint fields")
		}
		if err := validateSTUNServer(stunServer); err != nil {
			return err
		}
	case ProbeDelegatedDNS:
		if httpsURL != "" || stunServer != "" {
			return fmt.Errorf("delegated-DNS probe contains unrelated endpoint fields")
		}
		if err := validateDNSZone(dnsZone); err != nil {
			return err
		}
		if err := validateProbeURL(dnsResultURL, "delegated-DNS result"); err != nil {
			return err
		}
		if definition.MaxResponseBytes < 64 || definition.MaxResponseBytes > 64<<10 {
			return fmt.Errorf("delegated-DNS result limit is invalid")
		}
	}
	return nil
}

func (set ProbeSet) Validate() error {
	if set.SchemaVersion != ProbeSchemaVersion || !probeIDPattern.MatchString(set.ID) || set.Revision < 1 {
		return fmt.Errorf("probe-set identity is invalid")
	}
	if len(set.Definitions) < 1 || len(set.Definitions) > 12 {
		return fmt.Errorf("probe set must contain one to twelve definitions")
	}
	seenID := make(map[string]struct{}, len(set.Definitions))
	seenKind := make(map[ProbeKind]struct{}, len(set.Definitions))
	for index, definition := range set.Definitions {
		if err := definition.Validate(); err != nil {
			return fmt.Errorf("probe definition %d: %w", index, err)
		}
		key := fmt.Sprintf("%s@%d", definition.ID, definition.Revision)
		if _, exists := seenID[key]; exists {
			return fmt.Errorf("duplicate probe definition %q", key)
		}
		seenID[key] = struct{}{}
		if _, exists := seenKind[definition.Kind]; exists {
			return fmt.Errorf("probe set contains more than one %s definition", definition.Kind)
		}
		seenKind[definition.Kind] = struct{}{}
	}
	return nil
}

func NormalizeProbeSet(set ProbeSet) ProbeSet {
	sort.Slice(set.Definitions, func(i, j int) bool {
		if set.Definitions[i].Kind == set.Definitions[j].Kind {
			return set.Definitions[i].ID < set.Definitions[j].ID
		}
		return set.Definitions[i].Kind < set.Definitions[j].Kind
	})
	return set
}

func (set ProbeSet) Definition(kind ProbeKind) (ProbeDefinition, bool) {
	for _, definition := range set.Definitions {
		if definition.Kind == kind {
			return definition, true
		}
	}
	return ProbeDefinition{}, false
}

func validateProbeURL(raw, label string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return fmt.Errorf("%s probe URL is invalid", label)
	}
	if parsed.Scheme == "https" {
		return nil
	}
	if parsed.Scheme != "http" {
		return fmt.Errorf("%s probe must use HTTPS or loopback HTTP", label)
	}
	host := parsed.Hostname()
	ip := net.ParseIP(host)
	if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return fmt.Errorf("plain HTTP is allowed only for loopback probe fixtures")
	}
	return nil
}

func validateSTUNServer(raw string) error {
	if !strings.HasPrefix(strings.ToLower(raw), "stun:") || strings.ContainsAny(raw, "/?#@") {
		return fmt.Errorf("STUN probe endpoint is invalid")
	}
	hostPort := strings.TrimPrefix(raw, "stun:")
	host, port, err := net.SplitHostPort(hostPort)
	if err != nil || strings.TrimSpace(host) == "" {
		return fmt.Errorf("STUN probe requires an explicit host and port")
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return fmt.Errorf("STUN probe port is invalid")
	}
	if len(host) > 253 {
		return fmt.Errorf("STUN probe host is too long")
	}
	return nil
}

func validateDNSZone(raw string) error {
	raw = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(raw)), ".")
	if len(raw) > 253 || !dnsNamePattern.MatchString(raw) || net.ParseIP(raw) != nil {
		return fmt.Errorf("delegated-DNS probe zone is invalid")
	}
	return nil
}
