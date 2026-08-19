//go:build !windows

package kernel

func prepareManagedPackageRuntimeAccess(string) error {
	return nil
}

func releaseManagedPackageRuntimeAccess(string) error {
	return nil
}
