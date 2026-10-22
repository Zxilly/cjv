package projectcmd

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/cli/clictx"
	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/sdktools"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/spf13/cobra"
)

type doctorCheck struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
	Fix     string `json:"fix,omitempty"`
}

type doctorReport struct {
	SchemaVersion int           `json:"schema_version"`
	OK            bool          `json:"ok"`
	Checks        []doctorCheck `json:"checks"`
}

func (r doctorReport) Text() string {
	var b strings.Builder
	for _, check := range r.Checks {
		fmt.Fprintf(&b, "%s %s: %s\n", check.Status, check.ID, check.Message)
		if check.Path != "" {
			fmt.Fprintf(&b, "  %s\n", check.Path)
		}
		if check.Fix != "" {
			fmt.Fprintf(&b, "  Fix: %s\n", check.Fix)
		}
	}
	return b.String()
}

func (r *doctorReport) add(id, message, path, fix string, err error) {
	check := doctorCheck{ID: id, Status: "ok", Message: message, Path: path}
	if err != nil {
		r.OK = false
		check.Status, check.Message, check.Fix = "error", err.Error(), fix
	}
	r.Checks = append(r.Checks, check)
}

func newDoctorCommand(shared *clictx.Context) *cobra.Command {
	return &cobra.Command{
		Use: "doctor", Short: "Diagnose configuration, SDKs, and command entry points without changing them",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			report := diagnose(shared.Selector)
			if err := shared.Output.RenderTo(cmd.OutOrStdout(), report); err != nil {
				return err
			}
			if !report.OK {
				return &cjverr.ExitCodeError{Code: 1}
			}
			return nil
		},
	}
}

// diagnose deliberately avoids recovery, migration, lifecycle and resolver
// helpers that install missing SDKs. Every filesystem operation here is a read.
func diagnose(selector string) doctorReport {
	r := doctorReport{SchemaVersion: 1, OK: true, Checks: []doctorCheck{}}
	path, pathErr := config.SettingsPath()
	_, settings, settingsErr := config.LoadDefaultSettings()
	r.add("settings", "Settings loaded", path, "Correct the TOML settings in "+path, errors.Join(pathErr, settingsErr))
	if settingsErr == nil {
		_, err := config.ResolveAutoInstall(settings.AutoInstall)
		r.add("auto_install", "Automatic installation policy is valid", "", "Set CJV_AUTO_INSTALL to 0 or 1", err)
	}
	home, err := config.Home()
	if err == nil {
		var info os.FileInfo
		info, err = os.Stat(home)
		if err == nil && !info.IsDir() {
			err = fmt.Errorf("CJV_HOME is not a directory")
		}
	}
	r.add("home", "Data directory is accessible", home, "cjv init --yes --default-toolchain none", err)
	if home != "" {
		checkProxies(&r, home)
	}
	selected, err := toolchain.SelectActive(settings, settingsErr, selector)
	if err != nil {
		r.add("selection", "", "", "cjv default lts", err)
		return r
	}
	source := selected.Source.String()
	if selector != "" {
		source = "command line"
	}
	r.add("selection", selected.Name+" ("+source+")", "", "", nil)
	dir, display, parsed, err := toolchain.FindActiveDir(selected.Name)
	fix := "cjv install " + shellQuote(selected.Name) + " --reinstall"
	if parsed.IsCustom() || filepath.IsAbs(selected.Name) {
		fix = "Restore the custom SDK directory or select an installed SDK with cjv default <toolchain>"
	}
	r.add("installation", "Selected SDK directory exists", dir, fix, err)
	if err != nil {
		return r
	}
	tuple := ""
	recordRecovery := ""
	if !parsed.IsCustom() {
		record, err := toolchain.ReadInstallation(dir)
		if err != nil {
			// SDK mutations need a valid record too. Recover its provenance
			// before suggesting reinstall, component, or target commands.
			recordPath := filepath.Join(dir, ".cjv", "toolchain.toml")
			recordRecovery = "Restore " + recordPath + " from a trusted backup of the same SDK installation, then run cjv doctor again"
			fix = recordRecovery
		} else {
			tuple = record.Tuple
		}
		r.add("installation.record", record.Release, dir, fix, err)
	}
	for _, tool := range []string{"cjc", "cjpm"} {
		var binary string
		if tuple == "" {
			binary, err = sdktools.ResolveInstalledToolBinary(dir, tool)
		} else {
			binary, err = sdktools.ResolveInstalledToolBinaryForTuple(dir, tool, tuple)
		}
		if err == nil {
			err = checkExecutable(binary)
		}
		r.add("installation."+tool, tool+" is present", binary, fix, err)
	}
	checkProjectRequirements(&r, selected, display, dir, recordRecovery)
	return r
}

func checkProxies(r *doctorReport, home string) {
	bin := filepath.Join(home, "bin")
	managed := filepath.Join(bin, sdktools.CjvBinaryName())
	fix := "cjv init --yes --default-toolchain none; then reopen the shell"
	for _, tool := range []string{"cjc", "cjpm"} {
		expected := filepath.Join(bin, sdktools.PlatformBinaryName(tool))
		actual, err := exec.LookPath(tool)
		if err == nil && !samePath(actual, expected) {
			err = fmt.Errorf("PATH resolves %s to %s instead of %s", tool, actual, expected)
		}
		r.add("path."+tool, "PATH selects the cjv proxy", actual, fix, err)
		err = checkExecutable(expected)
		if err == nil {
			var same bool
			same, err = sameContents(managed, expected)
			if err == nil && !same {
				err = fmt.Errorf("proxy does not match the managed cjv binary")
			}
		}
		r.add("proxy."+tool, "Proxy matches the managed cjv binary", expected, fix, err)
	}
}

func checkProjectRequirements(r *doctorReport, selected config.ToolchainConfig, display, dir, recordRecovery string) {
	repair := func(command string) string {
		if recordRecovery != "" {
			return recordRecovery
		}
		return command
	}
	components, err := component.NormalizeList(selected.Components)
	r.add("project.components", "Component names are valid", "", "Correct components in cangjie-sdk.toml", err)
	installed, err := component.ListInstalled(dir)
	r.add("installation.components", "Component index is readable", dir, repair("cjv install "+shellQuote(display)+" --reinstall"), err)
	for _, name := range installed {
		if !slices.Contains(components, name) {
			components = append(components, name)
		}
	}
	for _, name := range components {
		err := checkComponentFiles(dir, display, name)
		r.add("component."+string(name), "Component installation record and files are present", dir, repair("cjv component add "+string(name)+" --toolchain "+shellQuote(display)+" --force"), err)
	}
	if len(selected.Targets) == 0 {
		return
	}
	host, err := toolchain.ReadHostTargets(dir)
	if err != nil {
		r.add("project.targets", "", dir, repair("cjv install "+shellQuote(display)+" --reinstall"), err)
		return
	}
	for _, environment := range selected.Targets {
		targetDir, _, err := host.FindTarget(environment)
		r.add("target."+environment, "Target matches the selected host release", targetDir, repair("cjv target add "+shellQuote(environment)+" --toolchain "+shellQuote(display)), err)
	}
}

func checkComponentFiles(dir, display string, name component.Name) error {
	paths, err := component.ReadManifest(dir, name)
	if err != nil {
		return err
	}
	roots, err := component.RootsFor(display)
	if err != nil {
		return err
	}
	spec, err := component.SpecFor(name)
	if err != nil {
		return err
	}
	root := spec.InstallRoot(roots)
	for _, relative := range paths {
		if !filepath.IsLocal(filepath.FromSlash(relative)) {
			return fmt.Errorf("component %s contains an invalid installed path %q", name, relative)
		}
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(relative))); err != nil {
			return err
		}
	}
	return nil
}

func checkExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", path)
	}
	if runtime.GOOS != "windows" && info.Mode()&0o111 == 0 {
		return fmt.Errorf("%s is not executable", path)
	}
	return nil
}

func samePath(a, b string) bool {
	a, b = config.NormalizePath(a), config.NormalizePath(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func sameContents(a, b string) (bool, error) {
	ai, err := os.Stat(a)
	if err != nil {
		return false, err
	}
	bi, err := os.Stat(b)
	if err != nil {
		return false, err
	}
	if os.SameFile(ai, bi) {
		return true, nil
	}
	if ai.Size() != bi.Size() || !ai.Mode().IsRegular() || !bi.Mode().IsRegular() {
		return false, nil
	}
	hash := func(path string) ([sha256.Size]byte, error) {
		var digest [sha256.Size]byte
		f, err := os.Open(path)
		if err != nil {
			return digest, err
		}
		defer f.Close() //nolint:errcheck
		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			return digest, err
		}
		copy(digest[:], h.Sum(nil))
		return digest, nil
	}
	ah, err := hash(a)
	if err != nil {
		return false, err
	}
	bh, err := hash(b)
	return ah == bh, err
}

func shellQuote(value string) string {
	if runtime.GOOS == "windows" {
		return "'" + strings.ReplaceAll(value, "'", "''") + "'"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
