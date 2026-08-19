//go:build !windows

package systemproxy

// Current is deliberately conservative outside Windows. Environment variables,
// desktop settings and PAC/WPAD are different contracts and are not guessed.
func Current() (Result, error) {
	return Result{Source: "unsupported", Reason: ReasonUnsupported}, nil
}
