package lifecycle

import (
	"context"
	"fmt"
	"path/filepath"

	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// InstallTargetsForToolchain installs missing cross SDKs for the release already
// inside the selected host. Proxy auto-install must not upgrade that host merely
// because a project requests a target that has not been installed yet.
func InstallTargetsForToolchain(ctx context.Context, input string, targets []string, opts Options) error {
	if len(targets) == 0 {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	targets, err := sdktarget.NormalizeList(targets)
	if err != nil {
		return err
	}
	d, err := OpenDistribution(opts)
	if err != nil {
		return err
	}
	dir, _, _, err := toolchain.FindActiveDir(input)
	if err != nil {
		return err
	}
	host, err := toolchain.InstalledRelease(dir)
	if err != nil {
		return err
	}
	if host.IsCustom() || host.Version == "" {
		return fmt.Errorf("toolchain %s has no release for cross SDK installation", input)
	}
	identity, err := toolchain.ParseToolchainName(filepath.Base(dir))
	if err != nil {
		return err
	}
	for _, target := range targets {
		tuple, err := d.TargetTuple(target)
		if err != nil {
			return err
		}
		rt, err := d.Resolve(ctx, host, tuple)
		if err != nil {
			return err
		}
		if err := installSelected(ctx, d, rt, identity.Version == "", false, false, opts); err != nil {
			return err
		}
	}
	return nil
}
