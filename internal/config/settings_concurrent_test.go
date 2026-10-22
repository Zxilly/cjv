package config

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettingsFileIndependentWritersMergeStaleSnapshots(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.toml")
	first, second := NewSettingsFile(path), NewSettingsFile(path)
	old, err := first.Load()
	require.NoError(t, err)
	_, err = second.Load()
	require.NoError(t, err)

	defaultName, projectName := "lts", "sts"
	_, err = first.Update(SettingsUpdate{DefaultToolchain: &defaultName, OverrideChanges: map[string]*string{"first": &projectName}})
	require.NoError(t, err)
	autoInstall := false
	_, err = second.Update(SettingsUpdate{AutoInstall: &autoInstall, OverrideChanges: map[string]*string{"second": &defaultName}})
	require.NoError(t, err)
	// A snapshot Save must merge only the caller's edits as well.
	old.AutoSelfUpdate = AutoSelfUpdateDisable
	old.Overrides["third"] = "nightly"
	require.NoError(t, first.Save(old))

	got, err := NewSettingsFile(path).Load()
	require.NoError(t, err)
	assert.Equal(t, "lts", got.DefaultToolchain)
	assert.False(t, got.AutoInstall)
	assert.Equal(t, AutoSelfUpdateDisable, got.AutoSelfUpdate)
	assert.Equal(t, map[string]string{"first": "sts", "second": "lts", "third": "nightly"}, got.Overrides)
}

func TestSettingsUndoPreservesConcurrentValuesAndFieldPresence(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvFallbackSettings, filepath.Join(dir, "fallback.toml"))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "fallback.toml"), []byte("auto_install = false\n"), 0o600))
	path := filepath.Join(dir, "settings.toml")
	first, second := NewSettingsFile(path), NewSettingsFile(path)
	initial, owned, concurrent := "lts", "sts", "nightly"
	_, err := first.Update(SettingsUpdate{OverrideChanges: map[string]*string{"removed": &initial}})
	require.NoError(t, err)
	on := true
	_, undo, err := first.UpdateWithUndo(SettingsUpdate{
		DefaultToolchain: &owned,
		AutoInstall:      &on,
		OverrideChanges:  map[string]*string{"same": &owned, "removed": nil},
	})
	require.NoError(t, err)
	_, err = second.Update(SettingsUpdate{
		DefaultToolchain: &concurrent,
		OverrideChanges:  map[string]*string{"same": &concurrent, "other": &initial},
	})
	require.NoError(t, err)
	require.NoError(t, undo())
	got, err := NewSettingsFile(path).Load()
	require.NoError(t, err)
	assert.Equal(t, concurrent, got.DefaultToolchain)
	assert.Equal(t, map[string]string{"same": concurrent, "other": initial, "removed": initial}, got.Overrides)
	assert.False(t, got.AutoInstall)
	var persisted map[string]any
	_, err = toml.DecodeFile(path, &persisted)
	require.NoError(t, err)
	assert.NotContains(t, persisted, "auto_install", "undo must restore inheritance, not copy the fallback value")
	// An already completed undo must not later erase a new matching value.
	_, err = second.Update(SettingsUpdate{AutoInstall: &on})
	require.NoError(t, err)
	require.NoError(t, undo())
	got, err = LoadSettings(path)
	require.NoError(t, err)
	assert.True(t, got.AutoInstall)
}

func TestSettingsFileProcessHelper(t *testing.T) {
	path := os.Getenv("CJV_TEST_SETTINGS_PATH")
	if path == "" {
		return
	}
	name := os.Getenv("CJV_TEST_SETTINGS_WRITER")
	sf := NewSettingsFile(path)
	_, err := sf.Load() // All workers deliberately hold the same stale snapshot.
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path+".ready-"+name, nil, 0o600))
	deadline := time.Now().Add(15 * time.Second)
	for {
		if _, err := os.Stat(path + ".start"); err == nil {
			break
		}
		require.True(t, time.Now().Before(deadline), "parent did not release writers")
		time.Sleep(10 * time.Millisecond)
	}
	_, err = sf.Mutate(func(latest *Settings) (SettingsUpdate, error) {
		// Each writer changes a distinct path and incorporates all existing
		// paths; serialized callbacks must observe previous completed writes.
		value := fmt.Sprintf("seen-%d", len(latest.Overrides))
		return SettingsUpdate{OverrideChanges: map[string]*string{name: &value}}, nil
	})
	require.NoError(t, err)
}

func TestSettingsFileConcurrentProcessesUseSettingsPathLock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "shared-settings.toml")
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	type worker struct {
		cmd    *exec.Cmd
		output bytes.Buffer
	}
	workers := make([]*worker, 4)
	for i := range workers {
		name := fmt.Sprint(i)
		w := &worker{cmd: exec.CommandContext(ctx, os.Args[0], "-test.run=^TestSettingsFileProcessHelper$")}
		w.cmd.Env = append(os.Environ(),
			"CJV_TEST_SETTINGS_PATH="+path,
			"CJV_TEST_SETTINGS_WRITER="+name,
			EnvHome+"="+filepath.Join(dir, "data-"+name),
			EnvFallbackSettings+"="+filepath.Join(dir, "no-fallback.toml"))
		w.cmd.Stdout, w.cmd.Stderr = &w.output, &w.output
		require.NoError(t, w.cmd.Start())
		workers[i] = w
	}
	require.Eventually(t, func() bool {
		ready, _ := filepath.Glob(path + ".ready-*")
		return len(ready) == len(workers)
	}, 10*time.Second, 10*time.Millisecond)
	require.NoError(t, os.WriteFile(path+".start", nil, 0o600))
	for _, w := range workers {
		require.NoError(t, w.cmd.Wait(), w.output.String())
	}
	got, err := LoadSettings(path)
	require.NoError(t, err)
	require.Len(t, got.Overrides, len(workers))
	values := make([]string, 0, len(got.Overrides))
	for _, value := range got.Overrides {
		values = append(values, value)
	}
	assert.ElementsMatch(t, []string{"seen-0", "seen-1", "seen-2", "seen-3"}, values)
}
