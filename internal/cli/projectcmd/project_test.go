package projectcmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/cli/clitest"
	"github.com/Zxilly/cjv/internal/cli/toolchaincmd"
	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/sdktools"
	"github.com/Zxilly/cjv/internal/testutil"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func isolateProject(t *testing.T) (string, string) {
	t.Helper()
	home, project := t.TempDir(), t.TempDir()
	configtest.Isolate(t, home)
	t.Setenv(config.EnvToolchain, "")
	t.Setenv(config.EnvFallbackSettings, filepath.Join(home, "missing-fallback.toml"))
	t.Setenv("CJV_OFFLINE", "1")
	t.Chdir(project)
	return home, project
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(body), 0o755))
}

func installedTracking(t *testing.T, home string) string {
	t.Helper()
	dir := filepath.Join(home, "toolchains", "lts")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, toolchain.WriteInstallation(dir, toolchain.Installation{Release: "lts-1.0.5"}))
	for _, name := range []string{"cjc", "cjpm"} {
		path, err := sdktools.ResolveToolBinary(dir, name)
		require.NoError(t, err)
		writeFile(t, path, "test SDK executable")
	}
	return dir
}

func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	result := map[string]string{}
	require.NoError(t, filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		var contents []byte
		if !entry.IsDir() {
			contents, err = os.ReadFile(path)
			if err != nil {
				return err
			}
		}
		// Windows directory-entry timestamps may lag behind file creation.
		// Presence and mode still catch directory mutations; file timestamps
		// also catch rewriting unchanged bytes.
		modified := int64(0)
		if !entry.IsDir() {
			modified = info.ModTime().UnixNano()
		}
		result[rel] = fmt.Sprintf("%s %d %x", info.Mode(), modified, sha256.Sum256(contents))
		return nil
	}))
	return result
}

func TestPinUsesInstalledTrackingAndPreservesProjectRequirements(t *testing.T) {
	home, project := isolateProject(t)
	installedTracking(t, home)
	path := filepath.Join(project, config.ToolchainFileName)
	writeFile(t, path, "[toolchain]\nchannel = 'lts'\ncomponents = ['stdx']\ntargets = ['ohos']\n[project]\nname = 'preserved'\n")
	before := snapshotTree(t, home)
	out, err := clitest.Execute(t, clitest.Root(Register), "pin", "--json")
	require.NoError(t, err)
	var result pinResult
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Equal(t, "lts-1.0.5", result.Toolchain)
	assert.False(t, result.Checked)
	assert.Equal(t, before, snapshotTree(t, home), "pin must not recover or install SDK state")
	file, err := config.ParseToolchainFile(path)
	require.NoError(t, err)
	assert.Equal(t, "lts-1.0.5", file.Toolchain.Channel)
	assert.Equal(t, []string{"stdx"}, file.Toolchain.Components)
	assert.Equal(t, []string{"ohos"}, file.Toolchain.Targets)
	var document map[string]any
	_, err = toml.DecodeFile(path, &document)
	require.NoError(t, err)
	assert.Equal(t, "preserved", document["project"].(map[string]any)["name"])
	beforeProject := snapshotTree(t, project)
	out, err = clitest.Execute(t, clitest.Root(Register), "pin", "--check", "--json")
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.True(t, result.Checked)
	assert.Equal(t, beforeProject, snapshotTree(t, project))
	assert.Equal(t, before, snapshotTree(t, home))
}

func TestPinResolvesMissingSDKWithoutInstalling(t *testing.T) {
	home, project := isolateProject(t)
	t.Setenv("CJV_OFFLINE", "0")
	server := testutil.ValidMockServer(t)
	sf, err := config.DefaultSettingsFile()
	require.NoError(t, err)
	url := server.URL + "/sdk-versions.json"
	_, err = sf.Update(config.SettingsUpdate{ManifestURL: &url})
	require.NoError(t, err)
	_, err = clitest.Execute(t, clitest.Root(Register), "pin", "lts")
	require.NoError(t, err)
	file, err := config.ParseToolchainFile(filepath.Join(project, config.ToolchainFileName))
	require.NoError(t, err)
	assert.Equal(t, "lts-1.0.5", file.Toolchain.Channel)
	assert.NoDirExists(t, filepath.Join(home, "toolchains"))
}

func TestPinCheckRejectsUnpinnedOrInvalidWithoutWrites(t *testing.T) {
	for _, body := range []string{
		"[toolchain]\nchannel = 'lts'\n",
		"[toolchain]\nchannel = 'lts-1.0'\n",
		"[toolchain]\nchannel = 'nightly-2026-10-22'\n",
		"[toolchain]\nchannel = 'lts-garbage'\n",
		"[toolchain]\nchannel = '1.0.5'\n",
		"[toolchain]\nchannel = 'custom-sdk'\n",
		"[toolchain]\nchannel = 'lts-1.0.5'\ncomponents = ['unknown']\n",
		"[toolchain]\nchannel = [\n",
	} {
		t.Run(body, func(t *testing.T) {
			home, project := isolateProject(t)
			writeFile(t, filepath.Join(project, config.ToolchainFileName), body)
			beforeHome, beforeProject := snapshotTree(t, home), snapshotTree(t, project)
			_, err := clitest.Execute(t, clitest.Root(Register), "pin", "--check")
			require.Error(t, err)
			assert.Equal(t, beforeHome, snapshotTree(t, home))
			assert.Equal(t, beforeProject, snapshotTree(t, project))
		})
	}
}

func TestPinRejectsCustomAndPreservesInvalidFile(t *testing.T) {
	i18n.Init("en")
	t.Cleanup(func() { i18n.Init("en") })
	home, project := isolateProject(t)
	installedTracking(t, home)
	path := filepath.Join(project, config.ToolchainFileName)
	writeFile(t, path, "[toolchain]\nchannel = [\n")
	before := snapshotTree(t, project)
	_, err := clitest.Execute(t, clitest.Root(Register), "pin", "lts")
	require.Error(t, err)
	assert.Equal(t, before, snapshotTree(t, project))
	writeFile(t, path, "[toolchain]\npath = "+fmt.Sprintf("%q", filepath.ToSlash(home))+"\ncomponents = ['stdx']\n")
	before = snapshotTree(t, project)
	_, err = clitest.Execute(t, clitest.Root(Register), "pin")
	require.ErrorContains(t, err, "path SDK")
	assert.Equal(t, before, snapshotTree(t, project))
	_, err = clitest.Execute(t, clitest.Root(Register), "pin", "lts")
	require.NoError(t, err)
	file, err := config.ParseToolchainFile(path)
	require.NoError(t, err)
	assert.Empty(t, file.Toolchain.Path)
	assert.Equal(t, "lts-1.0.5", file.Toolchain.Channel)
	assert.Equal(t, []string{"stdx"}, file.Toolchain.Components)
}

func setupHealthyDoctor(t *testing.T) (string, string, string) {
	t.Helper()
	home, project := isolateProject(t)
	dir := installedTracking(t, home)
	writeFile(t, filepath.Join(project, config.ToolchainFileName), "[toolchain]\nchannel = 'lts'\n")
	bin := filepath.Join(home, "bin")
	for _, tool := range []string{"cjv", "cjc", "cjpm"} {
		writeFile(t, filepath.Join(bin, sdktools.PlatformBinaryName(tool)), "test cjv executable")
	}
	t.Setenv("PATH", bin)
	writeFile(t, filepath.Join(home, "toolchains", ".cjv-stage-orphan", "sentinel"), "must not recover")
	return home, project, dir
}

func TestDoctorIsReadOnlyAndReportsOneJSONDocument(t *testing.T) {
	home, project, _ := setupHealthyDoctor(t)
	beforeHome, beforeProject := snapshotTree(t, home), snapshotTree(t, project)
	out, err := clitest.Execute(t, clitest.Root(Register), "doctor", "--json")
	require.NoError(t, err)
	var report doctorReport
	decoder := json.NewDecoder(bytes.NewBufferString(out))
	require.NoError(t, decoder.Decode(&report))
	assert.Equal(t, 1, report.SchemaVersion)
	assert.True(t, report.OK)
	assert.ErrorIs(t, decoder.Decode(&report), io.EOF)
	assert.Equal(t, beforeHome, snapshotTree(t, home))
	assert.Equal(t, beforeProject, snapshotTree(t, project))
}

func TestDoctorDiagnosesBrokenRecordToolsAndShadowedProxy(t *testing.T) {
	i18n.Init("en")
	t.Cleanup(func() { i18n.Init("en") })
	home, project, dir := setupHealthyDoctor(t)
	writeFile(t, filepath.Join(project, config.ToolchainFileName), "[toolchain]\nchannel = 'lts'\ntargets = ['ohos']\n")
	writeFile(t, filepath.Join(dir, ".cjv", "toolchain.toml"), "release = 'sts-1.0.5'\n")
	binary, err := sdktools.ResolveToolBinary(dir, "cjc")
	require.NoError(t, err)
	require.NoError(t, os.Remove(binary))
	shadow := t.TempDir()
	writeFile(t, filepath.Join(shadow, sdktools.PlatformBinaryName("cjc")), "other compiler")
	t.Setenv("PATH", shadow+string(os.PathListSeparator)+os.Getenv("PATH"))
	writeFile(t, filepath.Join(home, "bin", sdktools.PlatformBinaryName("cjpm")), "wrong proxy")
	require.NoError(t, component.WriteManifest(dir, component.Docs, []string{"missing.html"}))
	t.Setenv(config.EnvAutoInstall, "invalid")
	beforeHome, beforeProject := snapshotTree(t, home), snapshotTree(t, project)
	out, err := clitest.Execute(t, clitest.Root(Register), "doctor", "--json")
	var exit *cjverr.ExitCodeError
	require.ErrorAs(t, err, &exit)
	assert.Equal(t, 1, exit.Code)
	var report doctorReport
	decoder := json.NewDecoder(bytes.NewBufferString(out))
	require.NoError(t, decoder.Decode(&report))
	assert.False(t, report.OK)
	errors := map[string]doctorCheck{}
	for _, check := range report.Checks {
		if check.Status == "error" {
			errors[check.ID] = check
			assert.NotEmpty(t, check.Fix)
		}
	}
	for _, id := range []string{"path.cjc", "proxy.cjpm", "installation.record", "installation.cjc", "auto_install", "component.docs", "project.targets"} {
		assert.Contains(t, errors, id)
	}
	for _, id := range []string{"installation.record", "installation.cjc", "component.docs", "project.targets"} {
		assert.Contains(t, errors[id].Fix, "trusted backup", id)
		assert.Contains(t, errors[id].Fix, filepath.Join(dir, ".cjv", "toolchain.toml"), id)
		assert.NotContains(t, errors[id].Fix, "--reinstall", id)
		assert.NotContains(t, errors[id].Fix, "component add", id)
		assert.NotContains(t, errors[id].Fix, "target add", id)
	}
	assert.ErrorIs(t, decoder.Decode(&report), io.EOF)
	assert.Equal(t, beforeHome, snapshotTree(t, home))
	assert.Equal(t, beforeProject, snapshotTree(t, project))
}

func TestDoctorBrokenRecordRequiresTrustedBackup(t *testing.T) {
	i18n.Init("en")
	t.Cleanup(func() { i18n.Init("en") })
	for _, test := range []struct {
		name, contents string
	}{
		{name: "mismatched release", contents: "release = 'sts-1.0.5'\n"},
		{name: "invalid TOML", contents: "release = [\n"},
		{name: "missing record"},
	} {
		t.Run(test.name, func(t *testing.T) {
			home, project, dir := setupHealthyDoctor(t)
			recordPath := filepath.Join(dir, ".cjv", "toolchain.toml")
			backup, err := os.ReadFile(recordPath)
			require.NoError(t, err)
			if test.contents == "" {
				require.NoError(t, os.Remove(recordPath))
			} else {
				require.NoError(t, os.WriteFile(recordPath, []byte(test.contents), 0o600))
			}
			beforeHome, beforeProject := snapshotTree(t, home), snapshotTree(t, project)
			out, err := clitest.Execute(t, clitest.Root(Register), "doctor", "--json")
			var exit *cjverr.ExitCodeError
			require.ErrorAs(t, err, &exit)
			assert.Equal(t, 1, exit.Code)
			var report doctorReport
			require.NoError(t, json.Unmarshal([]byte(out), &report))
			record := findDoctorCheck(t, report, "installation.record")
			assert.Equal(t, "error", record.Status)
			assert.Contains(t, record.Fix, "trusted backup")
			assert.Contains(t, record.Fix, "same SDK installation")
			assert.Contains(t, record.Fix, recordPath)
			assert.Contains(t, record.Fix, "cjv doctor")
			assert.NotContains(t, record.Fix, "--reinstall")
			assert.Equal(t, beforeHome, snapshotTree(t, home))
			assert.Equal(t, beforeProject, snapshotTree(t, project))

			// Follow the manual recovery guidance using a backup from this exact
			// installation. Doctor must then accept the restored record.
			require.NoError(t, os.WriteFile(recordPath, backup, 0o600))
			out, err = clitest.Execute(t, clitest.Root(Register), "doctor", "--json")
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal([]byte(out), &report))
			assert.True(t, report.OK)
			assert.Equal(t, "ok", findDoctorCheck(t, report, "installation.record").Status)
		})
	}
}

func TestDoctorMissingCompilerCanReinstall(t *testing.T) {
	_, _, dir := setupHealthyDoctor(t)
	t.Setenv(config.EnvNoPathSetup, "1")
	t.Setenv(config.EnvDistServer, "")
	t.Setenv("CJV_OFFLINE", "0")
	server := testutil.ValidMockServer(t)
	sf, err := config.DefaultSettingsFile()
	require.NoError(t, err)
	manifest := server.URL + "/sdk-versions.json"
	_, err = sf.Update(config.SettingsUpdate{ManifestURL: &manifest})
	require.NoError(t, err)
	binary, err := sdktools.ResolveToolBinary(dir, "cjc")
	require.NoError(t, err)
	require.NoError(t, os.Remove(binary))
	out, err := clitest.Execute(t, clitest.Root(Register), "doctor", "--json")
	var exit *cjverr.ExitCodeError
	require.ErrorAs(t, err, &exit)
	var report doctorReport
	require.NoError(t, json.Unmarshal([]byte(out), &report))
	assert.Equal(t, "ok", findDoctorCheck(t, report, "installation.record").Status)
	compiler := findDoctorCheck(t, report, "installation.cjc")
	assert.Equal(t, "error", compiler.Status)
	assert.Equal(t, "cjv install "+shellQuote("lts")+" --reinstall", compiler.Fix)

	// Execute the suggested install through the real CLI and local mock source.
	_, err = clitest.Execute(t, clitest.Root(toolchaincmd.Register), "install", "lts", "--reinstall", "--quiet")
	require.NoError(t, err)
	require.NoError(t, checkExecutable(binary))
	record, err := toolchain.ReadInstallation(dir)
	require.NoError(t, err)
	assert.Equal(t, "lts-1.0.5", record.Release)
	out, err = clitest.Execute(t, clitest.Root(Register), "doctor", "--json")
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal([]byte(out), &report))
	assert.True(t, report.OK)
	assert.Equal(t, "ok", findDoctorCheck(t, report, "installation.cjc").Status)
}

func findDoctorCheck(t *testing.T, report doctorReport, id string) doctorCheck {
	t.Helper()
	for _, check := range report.Checks {
		if check.ID == id {
			return check
		}
	}
	t.Fatalf("doctor report has no %q check", id)
	return doctorCheck{}
}
