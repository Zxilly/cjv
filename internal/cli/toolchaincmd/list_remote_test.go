package toolchaincmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Zxilly/cjv/internal/config/configtest"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/testutil"
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
	hostKey, err := sdktarget.CurrentHostTuple("")
	require.NoError(t, err)
	hostOhos, err := sdktarget.CurrentTargetTuple("", "ohos")
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
	nightly := dist.ChannelInfo{
		Latest: "1.2.0-alpha.2",
		Versions: map[string]map[string]dist.DownloadInfo{
			"1.2.0-alpha.2": platformsForVersion(true),
			"1.2.0-alpha.1": platformsForVersion(false),
		},
	}
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	mux.HandleFunc("/sdk-versions.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(manifest)
	})
	mux.HandleFunc("/nightly.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(nightly)
	})
	return server
}

func setupListRemote(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	configtest.Isolate(t, home)
	require.NoError(t, config.EnsureDirs())
	server := listRemoteMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))
}

// newTestListRemote returns list-remote with its flag defaults applied.
func newTestListRemote() *listRemote {
	c := &listRemote{ctx: newTestContext()}
	c.command()
	return c
}

func newListRemoteCmd() (*cobra.Command, *bytes.Buffer) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	return cmd, buf
}

func TestListRemoteDefaultCurrentHost(t *testing.T) {
	c := newTestListRemote()
	setupListRemote(t)

	c.channel = "lts"

	cmd, buf := newListRemoteCmd()
	c.ctx.Output.SetJSONMode(true)

	require.NoError(t, c.run(cmd, nil))

	hostKey, err := sdktarget.CurrentHostTuple("")
	require.NoError(t, err)

	var got toolchainListRemoteResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, hostKey, got.Target)
	require.Len(t, got.Channels, 1)
	assert.Equal(t, "lts", got.Channels[0].Channel)
	assert.Equal(t, "1.0.5", got.Channels[0].Latest)
	assert.Equal(t, []string{"1.0.5", "1.0.4", "1.0.0"}, got.Channels[0].Versions)
}

func TestListRemoteTargetComposesPlatformKey(t *testing.T) {
	c := newTestListRemote()
	setupListRemote(t)

	c.channel = "lts"
	c.target = "ohos"

	cmd, buf := newListRemoteCmd()
	c.ctx.Output.SetJSONMode(true)

	require.NoError(t, c.run(cmd, nil))

	expected, err := sdktarget.CurrentTargetTuple("", "ohos")
	require.NoError(t, err)

	var got toolchainListRemoteResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, expected, got.Target)
	require.Len(t, got.Channels, 1)
	// Only 1.0.5 has an ohos build in the mock.
	assert.Equal(t, []string{"1.0.5"}, got.Channels[0].Versions)
}

func TestListRemoteTargetWithChannelAll_LtsFiltered(t *testing.T) {
	c := newTestListRemote()
	setupListRemote(t)

	c.target = "ohos"

	cmd, buf := newListRemoteCmd()
	c.ctx.Output.SetJSONMode(true)

	require.NoError(t, c.run(cmd, nil))

	var got toolchainListRemoteResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Len(t, got.Channels, 3)
	assert.Equal(t, []string{"1.0.5"}, got.Channels[0].Versions)
	assert.Empty(t, got.Channels[1].Versions, "STS has no ohos build in the mock")
	assert.Empty(t, got.Channels[1].Error, "missing builds is not a per-channel error")
	nightly := got.Channels[2]
	assert.Equal(t, "nightly", nightly.Channel)
	assert.Equal(t, []string{"1.2.0-alpha.2"}, nightly.Versions)
}

func TestListRemoteNightlyChannelFiltersByTarget(t *testing.T) {
	c := newTestListRemote()
	setupListRemote(t)

	c.channel = "nightly"
	c.target = "ohos"

	cmd, buf := newListRemoteCmd()
	c.ctx.Output.SetJSONMode(true)

	require.NoError(t, c.run(cmd, nil))
	var got toolchainListRemoteResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Len(t, got.Channels, 1)
	assert.Equal(t, []string{"1.2.0-alpha.2"}, got.Channels[0].Versions)
}

func TestListRemoteTargetAsHostKey_Rejected(t *testing.T) {
	c := newTestListRemote()
	setupListRemote(t)

	c.target = "linux-x64"

	cmd, _ := newListRemoteCmd()
	err := c.run(cmd, nil)
	require.Error(t, err)
}

func TestListRemoteLimitTruncates(t *testing.T) {
	c := newTestListRemote()
	setupListRemote(t)

	c.channel = "lts"
	c.limit = 2

	cmd, buf := newListRemoteCmd()
	c.ctx.Output.SetJSONMode(true)

	require.NoError(t, c.run(cmd, nil))

	var got toolchainListRemoteResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Len(t, got.Channels, 1)
	assert.Equal(t, []string{"1.0.5", "1.0.4"}, got.Channels[0].Versions)
}

func TestListRemoteUnknownChannelFlag(t *testing.T) {
	c := newTestListRemote()
	setupListRemote(t)

	c.channel = "weekly"

	cmd, _ := newListRemoteCmd()
	err := c.run(cmd, nil)
	require.Error(t, err)
	// Locale-independent: the rejected value is echoed in the message.
	assert.Contains(t, err.Error(), "weekly")
}

func TestListRemoteAllChannelsIncludeNightly(t *testing.T) {
	c := newTestListRemote()
	setupListRemote(t)

	cmd, buf := newListRemoteCmd()
	c.ctx.Output.SetJSONMode(true)

	require.NoError(t, c.run(cmd, nil))

	var got toolchainListRemoteResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Len(t, got.Channels, 3)
	assert.NotEmpty(t, got.Channels[0].Versions)
	assert.NotEmpty(t, got.Channels[1].Versions)
	assert.Equal(t, "nightly", got.Channels[2].Channel)
	assert.Equal(t, "1.2.0-alpha.2", got.Channels[2].Latest)
}

func TestListRemoteNightlyUsesDistServerManifest(t *testing.T) {
	c := newTestListRemote()
	home := t.TempDir()
	configtest.Isolate(t, home)
	require.NoError(t, config.EnsureDirs())

	server := testutil.SplitNightlyMockServer(t)
	settings := config.DefaultSettings()
	settings.DistServer = server.URL + "/corp/cjv"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	c.channel = "nightly"
	cmd, buf := newListRemoteCmd()
	c.ctx.Output.SetJSONMode(true)

	require.NoError(t, c.run(cmd, nil))
	var got toolchainListRemoteResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Len(t, got.Channels, 1)
	assert.Equal(t, "1.2.0-alpha.20260822010101", got.Channels[0].Latest)
	assert.Equal(t, []string{"1.2.0-alpha.20260822010101"}, got.Channels[0].Versions)

	c.ctx.Output.SetJSONMode(false)
	textCmd, textBuf := newListRemoteCmd()
	require.NoError(t, c.run(textCmd, nil))
	host, err := sdktarget.CurrentHostTuple("")
	require.NoError(t, err)
	assert.Contains(t, textBuf.String(), host)
}

func TestListRemoteAllPlatformsNightlyDistServerText(t *testing.T) {
	c := newTestListRemote()
	home := t.TempDir()
	configtest.Isolate(t, home)
	require.NoError(t, config.EnsureDirs())
	server := testutil.SplitNightlyMockServer(t)
	settings := config.DefaultSettings()
	settings.DistServer = server.URL + "/corp/cjv"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	c.channel = "nightly"
	c.allPlatforms = true
	cmd, buf := newListRemoteCmd()
	require.NoError(t, c.run(cmd, nil))

	host, err := sdktarget.CurrentHostTuple("")
	require.NoError(t, err)
	assert.Contains(t, buf.String(), host)
	assert.Contains(t, buf.String(), "1.2.0-alpha.20260822010101")
}

func TestListRemoteAllPlatforms(t *testing.T) {
	c := newTestListRemote()
	setupListRemote(t)

	c.allPlatforms = true
	c.channel = "lts"

	cmd, buf := newListRemoteCmd()
	c.ctx.Output.SetJSONMode(true)

	require.NoError(t, c.run(cmd, nil))

	var got toolchainListRemoteAllPlatformsResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.True(t, got.AllPlatforms)
	require.Len(t, got.Channels, 1)
	lts := got.Channels[0]
	assert.Equal(t, "lts", lts.Channel)
	assert.Equal(t, "1.0.5", lts.Latest)

	// Lexical sort places the bare host tuple before its environment-suffixed variants.
	hostTuple, err := sdktarget.CurrentHostTuple("")
	require.NoError(t, err)
	hostOhos, err := sdktarget.CurrentTargetTuple("", "ohos")
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

func TestListRemoteAllPlatforms_LimitPerPlatform(t *testing.T) {
	c := newTestListRemote()
	setupListRemote(t)

	c.allPlatforms = true
	c.channel = "lts"
	c.limit = 1

	cmd, buf := newListRemoteCmd()
	c.ctx.Output.SetJSONMode(true)

	require.NoError(t, c.run(cmd, nil))

	var got toolchainListRemoteAllPlatformsResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Len(t, got.Channels, 1)
	require.NotEmpty(t, got.Channels[0].Platforms)
	for _, p := range got.Channels[0].Platforms {
		assert.Len(t, p.Versions, 1, "platform %s should be limited to 1 version", p.Target)
	}
}

func TestListRemoteTextRendering_SinglePlatform(t *testing.T) {
	c := newTestListRemote()
	setupListRemote(t)

	c.channel = "lts"

	cmd, buf := newListRemoteCmd()
	require.NoError(t, c.run(cmd, nil))

	got := buf.String()
	hostKey, err := sdktarget.CurrentHostTuple("")
	require.NoError(t, err)
	assert.Contains(t, got, "lts")
	assert.Contains(t, got, "1.0.5")
	assert.Contains(t, got, hostKey)
}

func TestListRemoteTextRendering_AllPlatforms(t *testing.T) {
	c := newTestListRemote()
	setupListRemote(t)

	c.allPlatforms = true
	c.channel = "lts"

	cmd, buf := newListRemoteCmd()
	require.NoError(t, c.run(cmd, nil))

	got := buf.String()
	hostKey, err := sdktarget.CurrentHostTuple("")
	require.NoError(t, err)
	hostOhos, err := sdktarget.CurrentTargetTuple("", "ohos")
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

func TestListRemoteTargetMutuallyExclusiveWithAllPlatforms(t *testing.T) {
	configtest.Isolate(t, t.TempDir())
	require.NoError(t, config.EnsureDirs())

	_, err := execute(t, "toolchain", "list-remote", "--all-platforms", "--target", "ohos")
	require.Error(t, err)
	// cobra: "if any flags in the group [...] are set none of the others can be"
	assert.True(t,
		strings.Contains(err.Error(), "none of the others") ||
			strings.Contains(err.Error(), "mutually exclusive"),
		"unexpected error wording: %s", err.Error())
}

func TestListRemoteDoesNotWaitForLocalPublication(t *testing.T) {
	for _, allPlatforms := range []bool{false, true} {
		t.Run(map[bool]string{false: "host", true: "all-platforms"}[allPlatforms], func(t *testing.T) {
			setupListRemote(t)
			lock, err := toolchain.LockHome(t.Context())
			require.NoError(t, err)
			defer lock.Close() //nolint:errcheck
			c := newTestListRemote()
			c.channel = "lts"
			c.allPlatforms = allPlatforms
			cmd, output := newListRemoteCmd()
			result := make(chan error, 1)
			go func() { result <- c.run(cmd, nil) }()
			select {
			case err := <-result:
				require.NoError(t, err)
				require.Contains(t, output.String(), "1.0.5")
			case <-time.After(5 * time.Second):
				_ = lock.Close()
				<-result
				t.Fatal("remote catalog waited for local publication")
			}
		})
	}
}

func TestListRemoteDoesNotRecoverLocalInstallations(t *testing.T) {
	setupListRemote(t)
	home, err := config.Home()
	require.NoError(t, err)
	journal := filepath.Join(home, "toolchains", ".fstx-interrupted", "journal.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(journal), 0o755))
	require.NoError(t, os.WriteFile(journal, []byte("{"), 0o644))
	staging := filepath.Join(home, "toolchains", "sts.staging")
	require.NoError(t, os.MkdirAll(staging, 0o755))
	c := newTestListRemote()
	c.channel = "lts"
	cmd, output := newListRemoteCmd()
	require.NoError(t, c.run(cmd, nil))
	require.Contains(t, output.String(), "1.0.5")
	require.FileExists(t, journal)
	require.DirExists(t, staging)
}

func TestListRemoteDoesNotScanLocalInstallations(t *testing.T) {
	setupListRemote(t)
	home, err := config.Home()
	require.NoError(t, err)
	root := filepath.Join(home, "toolchains")
	require.NoError(t, os.Remove(root))
	require.NoError(t, os.WriteFile(root, []byte("not a directory"), 0o644))
	c := newTestListRemote()
	c.channel = "lts"
	cmd, output := newListRemoteCmd()
	require.NoError(t, c.run(cmd, nil))
	require.Contains(t, output.String(), "1.0.5")
	require.FileExists(t, root)
}

func TestListRemotePropagatesCancellation(t *testing.T) {
	for _, allPlatforms := range []bool{false, true} {
		t.Run(map[bool]string{false: "host", true: "all-platforms"}[allPlatforms], func(t *testing.T) {
			setupListRemote(t)
			c := newTestListRemote()
			c.channel = "lts"
			c.allPlatforms = allPlatforms
			cmd, output := newListRemoteCmd()
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			cmd.SetContext(ctx)
			require.ErrorIs(t, c.run(cmd, nil), context.Canceled)
			require.Empty(t, output.String())
		})
	}
}
