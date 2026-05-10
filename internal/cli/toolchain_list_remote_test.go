package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/cli/output"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// listRemoteMockServer serves only the manifest endpoint. validMockServer in
// install_test.go is single-version and ties URLs to a real SDK zip; we need
// multi-version, multi-tuple data with bogus URLs because list-remote only
// reads the manifest. The current host tuple always has builds for every
// fixture version so default-flag tests behave identically across host
// architectures.
func listRemoteMockServer(t *testing.T) *httptest.Server {
	t.Helper()

	const sha = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	mkInfo := func(name string) dist.DownloadInfo {
		return dist.DownloadInfo{Name: name + ".zip", SHA256: sha, URL: "https://example.invalid/" + name + ".zip"}
	}
	hostKey, err := dist.CurrentHostTuple("")
	require.NoError(t, err)
	hostOhos, err := dist.CurrentTargetTuple("", "ohos")
	require.NoError(t, err)

	platformsForVersion := func(includeOhos bool) map[string]dist.DownloadInfo {
		m := map[string]dist.DownloadInfo{
			hostKey:        mkInfo(hostKey + "-x"),
			"darwin-arm64": mkInfo("darwin-arm64-x"),
			"win32-x64":    mkInfo("win32-x64-x"),
		}
		if includeOhos {
			m[hostOhos] = mkInfo(hostOhos + "-x")
		}
		return m
	}

	var manifest dist.Manifest
	manifest.Channels.LTS = dist.ChannelInfo{
		Latest: "1.0.5",
		Versions: map[string]map[string]dist.DownloadInfo{
			"1.0.5": platformsForVersion(true),
			"1.0.4": platformsForVersion(false),
			"1.0.0": {hostKey: mkInfo(hostKey + "-1.0.0")},
		},
	}
	manifest.Channels.STS = dist.ChannelInfo{
		Latest: "1.1.0-beta.23",
		Versions: map[string]map[string]dist.DownloadInfo{
			"1.1.0-beta.23": {
				hostKey:     mkInfo(hostKey + "-beta"),
				"win32-x64": mkInfo("win32-x64-beta"),
			},
			"1.0.0": {hostKey: mkInfo(hostKey + "-1.0.0")},
		},
	}

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	mux.HandleFunc("/sdk-versions.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(manifest)
	})
	return server
}

func setupListRemote(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())
	server := listRemoteMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	settings.GitCodeAPIKey = "" // unset so nightly fetch returns the missing-key error
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))
}

// resetListRemoteFlags clears state between subtests because flags are
// package-level vars mutated by previous tests.
func resetListRemoteFlags() {
	toolchainListRemoteChannel = "all"
	toolchainListRemoteTarget = ""
	toolchainListRemoteAllPlatforms = false
	toolchainListRemoteLimit = 0
}

func newListRemoteCmd() (*cobra.Command, *bytes.Buffer) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	return cmd, buf
}

func TestRunToolchainListRemote_DefaultCurrentHost(t *testing.T) {
	setupListRemote(t)
	resetListRemoteFlags()
	toolchainListRemoteChannel = "lts"

	cmd, buf := newListRemoteCmd()
	output.SetJSONMode(true)
	t.Cleanup(func() { output.SetJSONMode(false) })

	require.NoError(t, runToolchainListRemote(cmd, nil))

	hostKey, err := dist.CurrentHostTuple("")
	require.NoError(t, err)

	var got toolchainListRemoteResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, hostKey, got.Target)
	require.Len(t, got.Channels, 1)
	assert.Equal(t, "lts", got.Channels[0].Channel)
	assert.Equal(t, "1.0.5", got.Channels[0].Latest)
	assert.Equal(t, []string{"1.0.5", "1.0.4", "1.0.0"}, got.Channels[0].Versions)
}

func TestRunToolchainListRemote_TargetComposesPlatformKey(t *testing.T) {
	setupListRemote(t)
	resetListRemoteFlags()
	toolchainListRemoteChannel = "lts"
	toolchainListRemoteTarget = "ohos"

	cmd, buf := newListRemoteCmd()
	output.SetJSONMode(true)
	t.Cleanup(func() { output.SetJSONMode(false) })

	require.NoError(t, runToolchainListRemote(cmd, nil))

	expected, err := dist.CurrentTargetTuple("", "ohos")
	require.NoError(t, err)

	var got toolchainListRemoteResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, expected, got.Target)
	require.Len(t, got.Channels, 1)
	// Only 1.0.5 has an ohos build in the mock.
	assert.Equal(t, []string{"1.0.5"}, got.Channels[0].Versions)
}

func TestRunToolchainListRemote_TargetWithChannelAll_LtsFiltered(t *testing.T) {
	setupListRemote(t)
	resetListRemoteFlags()
	toolchainListRemoteTarget = "ohos"

	cmd, buf := newListRemoteCmd()
	output.SetJSONMode(true)
	t.Cleanup(func() { output.SetJSONMode(false) })

	require.NoError(t, runToolchainListRemote(cmd, nil))

	var got toolchainListRemoteResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Len(t, got.Channels, 3)
	assert.Equal(t, []string{"1.0.5"}, got.Channels[0].Versions)
	assert.Empty(t, got.Channels[1].Versions, "STS has no ohos build in the mock")
	assert.Empty(t, got.Channels[1].Error, "missing builds is not a per-channel error")
	// Nightly tag is target-orthogonal: error must come from missing API key,
	// not from a 'target unsupported' check.
	nightly := got.Channels[2]
	assert.Equal(t, "nightly", nightly.Channel)
	assert.NotContains(t, nightly.Error, "target")
}

func TestRunToolchainListRemote_NightlyChannelWithTarget_NoTargetCheck(t *testing.T) {
	// nightly + --target should NOT special-case the target. Without an API
	// key the error surfaces from the GitCode call, not from a target check.
	setupListRemote(t)
	resetListRemoteFlags()
	toolchainListRemoteChannel = "nightly"
	toolchainListRemoteTarget = "ohos"

	cmd, _ := newListRemoteCmd()
	err := runToolchainListRemote(cmd, nil)
	require.Error(t, err, "without API key, the GitCode missing-key error should surface")
	assert.NotContains(t, err.Error(), "target")
}

func TestRunToolchainListRemote_TargetAsHostKey_Rejected(t *testing.T) {
	setupListRemote(t)
	resetListRemoteFlags()
	toolchainListRemoteTarget = "linux-x64"

	cmd, _ := newListRemoteCmd()
	err := runToolchainListRemote(cmd, nil)
	require.Error(t, err)
}

func TestRunToolchainListRemote_LimitTruncates(t *testing.T) {
	setupListRemote(t)
	resetListRemoteFlags()
	toolchainListRemoteChannel = "lts"
	toolchainListRemoteLimit = 2

	cmd, buf := newListRemoteCmd()
	output.SetJSONMode(true)
	t.Cleanup(func() { output.SetJSONMode(false) })

	require.NoError(t, runToolchainListRemote(cmd, nil))

	var got toolchainListRemoteResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Len(t, got.Channels, 1)
	assert.Equal(t, []string{"1.0.5", "1.0.4"}, got.Channels[0].Versions)
}

func TestRunToolchainListRemote_UnknownChannelFlag(t *testing.T) {
	setupListRemote(t)
	resetListRemoteFlags()
	toolchainListRemoteChannel = "weekly"

	cmd, _ := newListRemoteCmd()
	err := runToolchainListRemote(cmd, nil)
	require.Error(t, err)
	// Locale-independent: the rejected value is echoed in the message.
	assert.Contains(t, err.Error(), "weekly")
}

func TestRunToolchainListRemote_NightlyMissingKey_AllChannel(t *testing.T) {
	setupListRemote(t)
	resetListRemoteFlags()

	cmd, buf := newListRemoteCmd()
	output.SetJSONMode(true)
	t.Cleanup(func() { output.SetJSONMode(false) })

	require.NoError(t, runToolchainListRemote(cmd, nil), "missing nightly key must not fail the all-channel command")

	var got toolchainListRemoteResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Len(t, got.Channels, 3)
	assert.NotEmpty(t, got.Channels[0].Versions)
	assert.NotEmpty(t, got.Channels[1].Versions)
	assert.Equal(t, "nightly", got.Channels[2].Channel)
	assert.NotEmpty(t, got.Channels[2].Error)
}

func TestRunToolchainListRemote_NightlyMissingKey_ChannelExplicit_Errors(t *testing.T) {
	setupListRemote(t)
	resetListRemoteFlags()
	toolchainListRemoteChannel = "nightly"

	cmd, _ := newListRemoteCmd()
	err := runToolchainListRemote(cmd, nil)
	require.Error(t, err, "explicit --channel nightly must surface the missing-key error")
}

func TestRunToolchainListRemote_AllPlatforms(t *testing.T) {
	setupListRemote(t)
	resetListRemoteFlags()
	toolchainListRemoteAllPlatforms = true
	toolchainListRemoteChannel = "lts"

	cmd, buf := newListRemoteCmd()
	output.SetJSONMode(true)
	t.Cleanup(func() { output.SetJSONMode(false) })

	require.NoError(t, runToolchainListRemote(cmd, nil))

	var got toolchainListRemoteAllPlatformsResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.True(t, got.AllPlatforms)
	require.Len(t, got.Channels, 1)
	lts := got.Channels[0]
	assert.Equal(t, "lts", lts.Channel)
	assert.Equal(t, "1.0.5", lts.Latest)

	// Lexical sort places the bare host tuple before its environment-suffixed variants.
	hostTuple, err := dist.CurrentHostTuple("")
	require.NoError(t, err)
	hostOhos, err := dist.CurrentTargetTuple("", "ohos")
	require.NoError(t, err)

	tuples := make([]string, 0, len(lts.Platforms))
	for _, p := range lts.Platforms {
		tuples = append(tuples, p.Target)
	}
	assert.Contains(t, tuples, hostTuple)
	assert.Contains(t, tuples, hostOhos)
	assert.True(t, sort.StringsAreSorted(tuples), "target tuples must be lexically sorted: %v", tuples)

	for _, p := range lts.Platforms {
		switch p.Target {
		case hostTuple:
			assert.Equal(t, []string{"1.0.5", "1.0.4", "1.0.0"}, p.Versions)
		case hostOhos:
			assert.Equal(t, []string{"1.0.5"}, p.Versions)
		}
	}
}

func TestRunToolchainListRemote_AllPlatforms_LimitPerPlatform(t *testing.T) {
	setupListRemote(t)
	resetListRemoteFlags()
	toolchainListRemoteAllPlatforms = true
	toolchainListRemoteChannel = "lts"
	toolchainListRemoteLimit = 1

	cmd, buf := newListRemoteCmd()
	output.SetJSONMode(true)
	t.Cleanup(func() { output.SetJSONMode(false) })

	require.NoError(t, runToolchainListRemote(cmd, nil))

	var got toolchainListRemoteAllPlatformsResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Len(t, got.Channels, 1)
	require.NotEmpty(t, got.Channels[0].Platforms)
	for _, p := range got.Channels[0].Platforms {
		assert.Len(t, p.Versions, 1, "platform %s should be limited to 1 version", p.Target)
	}
}

func TestRunToolchainListRemote_AllPlatforms_NightlyOnly_NoManifestCall(t *testing.T) {
	// A 500 from /sdk-versions.json proves the manifest endpoint is never hit
	// when only nightly is requested.
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	mux.HandleFunc("/sdk-versions.json", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "should not be called", http.StatusInternalServerError)
	})

	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	resetListRemoteFlags()
	toolchainListRemoteAllPlatforms = true
	toolchainListRemoteChannel = "nightly"

	cmd, _ := newListRemoteCmd()
	err := runToolchainListRemote(cmd, nil)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "manifest")
}

func TestRunToolchainListRemote_TextRendering_SinglePlatform(t *testing.T) {
	setupListRemote(t)
	resetListRemoteFlags()
	toolchainListRemoteChannel = "lts"

	cmd, buf := newListRemoteCmd()
	require.NoError(t, runToolchainListRemote(cmd, nil))

	got := buf.String()
	hostKey, err := dist.CurrentHostTuple("")
	require.NoError(t, err)
	assert.Contains(t, got, "lts")
	assert.Contains(t, got, "1.0.5")
	assert.Contains(t, got, hostKey)
}

func TestRunToolchainListRemote_TextRendering_AllPlatforms(t *testing.T) {
	setupListRemote(t)
	resetListRemoteFlags()
	toolchainListRemoteAllPlatforms = true
	toolchainListRemoteChannel = "lts"

	cmd, buf := newListRemoteCmd()
	require.NoError(t, runToolchainListRemote(cmd, nil))

	got := buf.String()
	hostKey, err := dist.CurrentHostTuple("")
	require.NoError(t, err)
	hostOhos, err := dist.CurrentTargetTuple("", "ohos")
	require.NoError(t, err)

	assert.Contains(t, got, "lts")
	assert.Contains(t, got, hostKey)
	assert.Contains(t, got, hostOhos)
	assert.Contains(t, got, "1.0.5")
	assert.Contains(t, got, "1.0.4")
}

func TestParseListRemoteChannel(t *testing.T) {
	cases := []struct {
		in          string
		wantChan    toolchain.Channel
		wantAll     bool
		wantErr     bool
		errContains string
	}{
		{"all", toolchain.UnknownChannel, true, false, ""},
		{"  ALL  ", toolchain.UnknownChannel, true, false, ""},
		{"lts", toolchain.LTS, false, false, ""},
		{"sts", toolchain.STS, false, false, ""},
		{"nightly", toolchain.Nightly, false, false, ""},
		{"weekly", toolchain.UnknownChannel, false, true, "weekly"},
		{"", toolchain.UnknownChannel, false, true, ""},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			ch, all, err := parseListRemoteChannel(tc.in)
			if tc.wantErr {
				require.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantChan, ch)
			assert.Equal(t, tc.wantAll, all)
		})
	}
}

func TestRunToolchainListRemote_TargetMutuallyExclusiveWithAllPlatforms(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())
	resetListRemoteFlags()

	rootCmd.SetArgs([]string{"toolchain", "list-remote", "--all-platforms", "--target", "ohos"})
	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetErr(&bytes.Buffer{})
	err := rootCmd.Execute()
	require.Error(t, err)
	// cobra: "if any flags in the group [...] are set none of the others can be"
	assert.True(t,
		strings.Contains(err.Error(), "none of the others") ||
			strings.Contains(err.Error(), "mutually exclusive"),
		"unexpected error wording: %s", err.Error())
}
