//go:build !windows

package lifecycle

func pathHasReparsePoint(string) (bool, error) { return false, nil }
