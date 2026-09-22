package selfmgmt

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Zxilly/cjv/internal/cli/output"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelfUpdateReportsAppliedVersionAfterProxyRefresh(t *testing.T) {
	for _, blocked := range []bool{false, true} {
		t.Run(fmt.Sprintf("proxy-blocked=%t", blocked), func(t *testing.T) {
			home := t.TempDir()
			config.IsolateForTest(t, home)
			t.Setenv(config.EnvMaxRetries, "0")
			binDir := filepath.Join(home, "bin")
			require.NoError(t, os.MkdirAll(binDir, 0o755))
			managed := filepath.Join(binDir, proxy.CjvBinaryName())
			require.NoError(t, os.WriteFile(managed, []byte("old executable"), 0o755))
			if blocked {
				obstruction := filepath.Join(binDir, proxy.PlatformBinaryName("cjc"))
				require.NoError(t, os.Mkdir(obstruction, 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(obstruction, "keep"), []byte("obstruction"), 0o644))
			}
			updateURL := serveManagedUpdate(t)
			previousJSON := output.IsJSON()
			output.SetJSONMode(true)
			t.Cleanup(func() { output.SetJSONMode(previousJSON) })
			var stdout, stderr bytes.Buffer
			cmd := NewSelfCommand("1.0.0", updateURL)
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs([]string{"update"})

			// Catch progress or outcome text that bypasses the JSON renderer.
			leaked, err := os.CreateTemp(t.TempDir(), "stdout-*")
			require.NoError(t, err)
			originalStdout := os.Stdout
			os.Stdout = leaked
			t.Cleanup(func() {
				os.Stdout = originalStdout
				_ = leaked.Close()
			})
			err = cmd.Execute()
			installed, readErr := os.ReadFile(managed)
			require.NoError(t, readErr)
			assert.Equal(t, "new executable", string(installed))
			if blocked {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "1.1.0")
				var renameErr *os.LinkError
				require.ErrorAs(t, err, &renameErr, "the proxy failure must remain inspectable")
				require.ErrorIs(t, output.RenderErrorTo(&stdout, &stderr, err), err)
				var envelope struct {
					Error struct {
						Code    string `json:"code"`
						Details struct {
							Version string `json:"version"`
							Updated bool   `json:"updated"`
							Status  string `json:"status"`
							Phase   string `json:"phase"`
						} `json:"details"`
					} `json:"error"`
				}
				require.NoError(t, json.Unmarshal(stdout.Bytes(), &envelope), "emit exactly one JSON document")
				assert.Equal(t, "SELF_UPDATE_FINALIZATION_FAILED", envelope.Error.Code)
				assert.Equal(t, "1.1.0", envelope.Error.Details.Version)
				assert.True(t, envelope.Error.Details.Updated)
				assert.Equal(t, "updated", envelope.Error.Details.Status)
				assert.Equal(t, "proxy-refresh", envelope.Error.Details.Phase)
			} else {
				require.NoError(t, err)
				var result UpdateResult
				require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
				assert.Equal(t, "1.1.0", result.Version)
				assert.True(t, result.Updated)
				for _, tool := range proxy.AllProxyTools() {
					data, readErr := os.ReadFile(filepath.Join(binDir, proxy.PlatformBinaryName(tool)))
					require.NoError(t, readErr)
					assert.Equal(t, "new executable", string(data), "proxy %s must use the replacement binary", tool)
				}
			}
			data, readErr := os.ReadFile(leaked.Name())
			require.NoError(t, readErr)
			assert.Empty(t, string(data))
			assert.Empty(t, stderr.String())
		})
	}
}

type managedUpdateTransport func(*http.Request) (*http.Response, error)

func (f managedUpdateTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// The server supports both release discovery adapters. GitHub's fixed endpoint
// is redirected locally; mirror builds use the local GitCode-shaped URL.
func serveManagedUpdate(t *testing.T) string {
	t.Helper()
	var archive bytes.Buffer
	w := zip.NewWriter(&archive)
	for _, binary := range []string{"cjv", "cjv-mirror"} {
		f, err := w.Create(proxy.PlatformBinaryName(binary))
		require.NoError(t, err)
		_, err = f.Write([]byte("new executable"))
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())
	digest := fmt.Sprintf("%x", sha256.Sum256(archive.Bytes()))
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	assetNames := []string{
		fmt.Sprintf("cjv_%s_%s%s", runtime.GOOS, runtime.GOARCH, ext),
		fmt.Sprintf("cjv-mirror_%s_%s%s", runtime.GOOS, runtime.GOARCH, ext),
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/owner/repo/releases/latest":
			assets := []map[string]string{{"name": "checksums.txt", "browser_download_url": "http://" + r.Host + "/checksums.txt"}}
			for _, name := range assetNames {
				assets = append(assets, map[string]string{"name": name, "browser_download_url": "http://" + r.Host + "/" + name})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"tag_name": "v1.1.0", "assets": assets})
		case "/owner/repo/releases/latest":
			http.Redirect(w, r, "/owner/repo/releases/tag/v1.1.0", http.StatusFound)
		default:
			switch path.Base(r.URL.Path) {
			case "checksums.txt":
				for _, name := range assetNames {
					_, _ = fmt.Fprintf(w, "%s  %s\n", digest, name)
				}
			case assetNames[0], assetNames[1]:
				_, _ = w.Write(archive.Bytes())
			default:
				http.NotFound(w, r)
			}
		}
	}))
	t.Cleanup(server.Close)
	target, err := url.Parse(server.URL)
	require.NoError(t, err)
	client := dist.HTTPClient()
	previousTransport := client.Transport
	client.Transport = managedUpdateTransport(func(req *http.Request) (*http.Response, error) {
		clone := req.Clone(req.Context())
		if clone.URL.Host == "api.github.com" {
			clone.URL.Scheme, clone.URL.Host = target.Scheme, target.Host
		}
		if clone.URL.Host != target.Host {
			return nil, fmt.Errorf("unexpected non-local update URL: %s", clone.URL)
		}
		return http.DefaultTransport.RoundTrip(clone)
	})
	t.Cleanup(func() { client.Transport = previousTransport })
	return server.URL + "/owner/repo/releases"
}
