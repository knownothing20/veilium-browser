//go:build !windows

package localrecovery

func pathHasReparsePoint(string) (bool, error) { return false, nil }
