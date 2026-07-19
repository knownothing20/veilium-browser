package networkevidence

import (
	"net"
	"strings"
)

func ReconcileObservations(observations []Observation) []Observation {
	result := append([]Observation(nil), observations...)
	exitIP := ""
	for _, observation := range result {
		if observation.ProbeKind == ProbeExitIP && observation.Status == ObservationPassed && len(observation.Values) == 1 {
			exitIP = normalizedIP(observation.Values[0])
			break
		}
	}
	for index := range result {
		switch result[index].ProbeKind {
		case ProbeWebRTCSTUN:
			result[index] = reconcileWebRTC(result[index], exitIP)
		case ProbeDelegatedDNS:
			result[index] = reconcileDNS(result[index])
		}
	}
	return result
}

func reconcileWebRTC(observation Observation, exitIP string) Observation {
	if observation.Status == ObservationUnavailable || observation.Status == ObservationSkipped {
		return observation
	}
	publicIPs := make([]string, 0, 4)
	for _, value := range observation.Values {
		if !strings.HasPrefix(value, "public-ip:") {
			continue
		}
		if ip := normalizedIP(strings.TrimPrefix(value, "public-ip:")); ip != "" {
			publicIPs = append(publicIPs, ip)
		}
	}
	publicIPs = sortedUnique(publicIPs)
	if len(publicIPs) == 0 {
		if observation.Status == ObservationPassed {
			observation.Status = ObservationPartial
		}
		if observation.ReasonCode == "" {
			observation.ReasonCode = "stun-no-public-address"
		}
		if observation.Detail == "" {
			observation.Detail = "The controlled STUN exchange did not expose a reflexive or relay public address."
		}
		return observation
	}
	if exitIP == "" {
		observation.Status = ObservationPartial
		observation.ReasonCode = "exit-ip-baseline-unavailable"
		observation.Detail = "STUN returned a public address, but no browser exit-IP baseline was available for comparison."
		return observation
	}
	for _, publicIP := range publicIPs {
		if publicIP == exitIP {
			continue
		}
		observation.Status = ObservationFailed
		observation.Expected = exitIP
		observation.ReasonCode = "webrtc-exit-ip-mismatch"
		observation.Detail = "The WebRTC/STUN public address differs from the browser-observed exit IP."
		return observation
	}
	observation.Status = ObservationPassed
	observation.Expected = exitIP
	observation.ReasonCode = ""
	observation.Detail = "WebRTC/STUN public addresses match the browser-observed exit IP."
	return observation
}

func reconcileDNS(observation Observation) Observation {
	if observation.Status == ObservationUnavailable || observation.Status == ObservationSkipped {
		return observation
	}
	seen := false
	for _, value := range observation.Values {
		if value == "seen:true" {
			seen = true
			break
		}
	}
	if seen {
		observation.Status = ObservationPassed
		observation.ReasonCode = ""
		if observation.Detail == "" {
			observation.Detail = "The delegated DNS probe observed the one-time browser query."
		}
		return observation
	}
	observation.Status = ObservationPartial
	if observation.ReasonCode == "" {
		observation.ReasonCode = "dns-query-not-seen"
	}
	if observation.Detail == "" {
		observation.Detail = "The delegated DNS probe did not observe the one-time query before the deadline."
	}
	return observation
}

func normalizedIP(raw string) string {
	ip := net.ParseIP(strings.TrimSpace(raw))
	if ip == nil {
		return ""
	}
	return ip.String()
}
