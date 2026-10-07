package cli

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateCancellationWhileHomeLocked(t *testing.T) {
	for _, args := range [][]string{
		{"update", "sts"},
		{"update", "--no-self-update"},
		{"update"},
	} {
		t.Run(fmt.Sprint(args), func(t *testing.T) {
			home := t.TempDir()
			config.IsolateForTest(t, home)
			t.Setenv(config.EnvNoPathSetup, "1")
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				http.NotFound(w, r)
			}))
			t.Cleanup(server.Close)
			sf, err := config.DefaultSettingsFile()
			require.NoError(t, err)
			manifest := server.URL + "/manifest.json"
			auto := config.AutoSelfUpdateEnable
			_, err = sf.Update(config.SettingsUpdate{ManifestURL: &manifest, AutoSelfUpdate: &auto})
			require.NoError(t, err)
			lock, err := toolchain.LockHome(t.Context())
			require.NoError(t, err)
			var releaseOnce sync.Once
			release := func() { releaseOnce.Do(func() { _ = lock.Close() }) }
			t.Cleanup(release)
			// A regression must fail promptly rather than hanging the test suite
			// forever inside recovery with a background context.
			fallback := time.AfterFunc(2*time.Second, release)
			defer fallback.Stop()
			ctx, cancel := context.WithTimeout(t.Context(), 75*time.Millisecond)
			defer cancel()
			app := newApplication("1.0.0", server.URL+"/self-update")
			app.rootCmd.SetContext(ctx)
			start := time.Now()
			_, err = executeWithOutput(t, app, args)
			require.ErrorIs(t, err, context.DeadlineExceeded)
			assert.Less(t, time.Since(start), time.Second, "cancellation must not wait for another operation's home lock")
			assert.Zero(t, requests.Load(), "canceled updates must not start distribution or self-update requests")
			assert.NoDirExists(t, filepath.Join(home, "bin"), "auto-self-update must not bootstrap a managed executable after cancellation")
		})
	}
}
