package lifecycle

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// Compare actual snapshot work with the original exclusive critical section.
// lookup-ns/op measures an unrelated SDK's startup lookup, not snapshot speed.
func BenchmarkActiveLookupDuringHistorySnapshot(b *testing.B) {
	for _, stable := range []bool{false, true} {
		name := "exclusive-baseline"
		if stable {
			name = "stable-read-phase"
		}
		b.Run(name, func(b *testing.B) {
			home := b.TempDir()
			b.Setenv(config.EnvHome, home)
			b.Setenv(config.EnvConfigFile, "")
			config.SetUserHomeDirOverrideForTest(home)
			config.ResetDefaultSettingsFileCache()
			b.Cleanup(func() {
				config.SetUserHomeDirOverrideForTest("")
				config.ResetDefaultSettingsFileCache()
			})
			must := func(err error) {
				b.Helper()
				if err != nil {
					b.Fatal(err)
				}
			}
			must(config.EnsureDirs())
			sdk := filepath.Join(home, config.ToolchainsSubdir, "sts")
			must(os.MkdirAll(sdk, 0o755))
			must(os.Mkdir(filepath.Join(home, config.ToolchainsSubdir, "custom"), 0o755))
			payload := bytes.Repeat([]byte("snapshot"), 512*1024)
			for i := range 4 {
				must(os.WriteFile(filepath.Join(sdk, fmt.Sprintf("payload-%d", i)), payload, 0o644))
			}
			record := toolchain.Installation{Release: "sts-1.0.0", Tuple: "win32-x64"}
			must(toolchain.WriteInstallation(sdk, record))
			lock, err := toolchain.LockHome(b.Context())
			must(err)
			must(lock.Recover())
			must(lock.MigrateLegacy())
			must(lock.Close())
			members := []*groupMember{{identity: "sts", installed: installationSnapshot{record: &record}}}
			var waited time.Duration
			b.SetBytes(int64(len(payload) * 4))
			b.ResetTimer()
			for range b.N {
				lock, err := toolchain.LockHome(b.Context())
				must(err)
				must(lock.Recover())
				started := make(chan struct{})
				type lookupResult struct {
					wait time.Duration
					err  error
				}
				done := make(chan lookupResult, 1)
				go func() {
					<-started
					start := time.Now()
					_, _, err := toolchain.PrepareActive(b.Context(), "custom", nil)
					done <- lookupResult{wait: time.Since(start), err: err}
				}()
				var snapshot *preparedHistory
				work := func() error {
					close(started)
					var err error
					snapshot, err = prepareHistorySnapshot(b.Context(), home, "sts", record.Tuple, members)
					return err
				}
				if stable {
					err = lock.WithStableReads(b.Context(), work)
				} else {
					err = work()
				}
				closeErr := lock.Close()
				lookup := <-done
				if snapshot != nil {
					snapshot.cleanup(nil)
				}
				must(errors.Join(err, closeErr, lookup.err))
				waited += lookup.wait
			}
			b.ReportMetric(float64(waited.Nanoseconds())/float64(b.N), "lookup-ns/op")
		})
	}
}
