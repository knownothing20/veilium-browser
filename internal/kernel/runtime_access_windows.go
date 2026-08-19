//go:build windows

package kernel

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/windows"
)

const managedPackageRuntimeAccessTimeout = 30 * time.Second

func prepareManagedPackageRuntimeAccess(packageRoot string) error {
	packageRoot, err := validatedManagedPackageRoot(packageRoot)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), managedPackageRuntimeAccessTimeout)
	defer cancel()

	// Chromium's Windows sandbox launches restricted and AppContainer processes.
	// The managed package stays byte-for-byte verified; these stable system SIDs
	// receive read/execute access only, while profile and credential roots remain
	// private and are never passed to this function.
	principals := []string{
		"*S-1-5-12:(OI)(CI)(RX)",
		"*S-1-15-2-1:(OI)(CI)(RX)",
		"*S-1-15-2-2:(OI)(CI)(RX)",
	}
	for _, principal := range principals {
		if err := runICACLS(ctx, packageRoot, "/grant", principal, "/T", "/Q"); err != nil {
			return fmt.Errorf("grant %s read/execute access: %w", principal, err)
		}
	}

	currentUserSID, err := currentProcessUserSID()
	if err != nil {
		return err
	}
	// Add an inherit-only ACE first so Windows propagates it without locking the
	// root while icacls is still traversing. The second ACE protects the root.
	// Generic write does not include DELETE/DELETE_CHILD, so managed uninstall
	// remains possible after the explicit release step below.
	for _, deniedWrite := range []string{
		"*" + currentUserSID + ":(OI)(CI)(IO)(WD,AD,WEA)",
		"*" + currentUserSID + ":(WD,AD,WEA)",
	} {
		if err := runICACLS(ctx, packageRoot, "/deny", deniedWrite, "/Q"); err != nil {
			return fmt.Errorf("deny current-user package write access: %w", err)
		}
	}
	return nil
}

func releaseManagedPackageRuntimeAccess(packageRoot string) error {
	packageRoot, err := validatedManagedPackageRoot(packageRoot)
	if err != nil {
		return err
	}
	currentUserSID, err := currentProcessUserSID()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), managedPackageRuntimeAccessTimeout)
	defer cancel()
	if err := runICACLS(ctx, packageRoot, "/remove:d", "*"+currentUserSID, "/Q"); err != nil {
		return fmt.Errorf("release current-user package write protection: %w", err)
	}
	return nil
}

func validatedManagedPackageRoot(packageRoot string) (string, error) {
	packageRoot = strings.TrimSpace(packageRoot)
	if packageRoot == "" || !filepath.IsAbs(packageRoot) {
		return "", fmt.Errorf("managed package root must be an absolute path")
	}
	packageRoot = filepath.Clean(packageRoot)
	info, err := os.Lstat(packageRoot)
	if err != nil {
		return "", fmt.Errorf("inspect managed package root: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", fmt.Errorf("managed package root must be a real directory")
	}
	return packageRoot, nil
}

func currentProcessUserSID() (string, error) {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return "", fmt.Errorf("resolve current user for managed package access: %w", err)
	}
	sid := strings.TrimSpace(user.User.Sid.String())
	if sid == "" {
		return "", fmt.Errorf("current user SID is empty")
	}
	return sid, nil
}

func runICACLS(ctx context.Context, packageRoot string, arguments ...string) error {
	command := exec.CommandContext(ctx, "icacls.exe", append([]string{packageRoot}, arguments...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}
