package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettingsFile_LoadCachesResult(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.toml")
	sf := NewSettingsFile(path)

	s1, err := sf.Load()
	if err != nil {
		t.Fatal(err)
	}
	s2, err := sf.Load()
	if err != nil {
		t.Fatal(err)
	}
	// Should be equal but not the same pointer (copy)
	if s1 == s2 {
		t.Error("Load should return copies, not the same pointer")
	}
	if s1.ManifestURL != s2.ManifestURL {
		t.Error("cached results should be equal")
	}
}

func TestSettingsFile_SaveUpdatesCacheAndDisk(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.toml")
	sf := NewSettingsFile(path)

	s, _ := sf.Load()
	s.DefaultToolchain = "lts-1.0.5"
	if err := sf.Save(s); err != nil {
		t.Fatal(err)
	}

	// Cache should reflect change
	s2, _ := sf.Load()
	if s2.DefaultToolchain != "lts-1.0.5" {
		t.Error("cache should reflect saved value")
	}

	// Disk should also reflect change
	sf2 := NewSettingsFile(path)
	s3, _ := sf2.Load()
	if s3.DefaultToolchain != "lts-1.0.5" {
		t.Error("disk should reflect saved value")
	}
}

func TestSettingsFile_Invalidate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.toml")
	sf := NewSettingsFile(path)

	sf.Load()
	sf.Invalidate()

	// Modify file on disk directly
	os.WriteFile(path, []byte(`default_toolchain = "sts-0.58.0"`+"\n"), 0o644)

	s, _ := sf.Load()
	if s.DefaultToolchain != "sts-0.58.0" {
		t.Error("after invalidate, should re-read from disk")
	}
}

func TestSettingsFile_ConcurrentAccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.toml")
	sf := NewSettingsFile(path)

	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			sf.Load()
		})
	}
	wg.Wait()
}

func TestSettingsFileSavePreservesAndRestoresProvenance(t *testing.T) {
	dir := t.TempDir()
	fallback := filepath.Join(dir, "fallback.toml")
	t.Setenv(EnvFallbackSettings, fallback)
	require.NoError(t, os.WriteFile(fallback, []byte("default_toolchain = 'lts-1.0.5'\nauto_install = false\n"), 0o600))
	sf := NewSettingsFile(filepath.Join(dir, "settings.toml"))
	before, err := sf.Load()
	require.NoError(t, err)

	changed := copySettings(before)
	changed.Overrides["project"] = "sts"
	require.NoError(t, sf.Save(changed))
	var stored map[string]any
	_, err = toml.DecodeFile(sf.Path(), &stored)
	require.NoError(t, err)
	assert.NotContains(t, stored, "default_toolchain")
	assert.NotContains(t, stored, "auto_install")
	assert.Contains(t, stored, "overrides")

	// Rollback restores field presence as well as the values seen before the
	// operation, so a later administrator change is still inherited.
	require.NoError(t, sf.Save(before))
	require.NoError(t, os.WriteFile(fallback, []byte("default_toolchain = 'lts-1.0.6'\nauto_install = true\n"), 0o600))
	sf.Invalidate()
	restored, err := sf.Load()
	require.NoError(t, err)
	assert.Empty(t, restored.Overrides)
	assert.Equal(t, "lts-1.0.6", restored.DefaultToolchain)
	assert.True(t, restored.AutoInstall)
}

func TestSettingsFileUpdateTracksExplicitChoices(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvFallbackSettings, filepath.Join(dir, "missing-fallback.toml"))
	t.Setenv(EnvDistServer, "https://temporary.example/cjv")
	sf := NewSettingsFile(filepath.Join(dir, "settings.toml"))
	autoInstall := true // Equal to the built-in default, but explicitly chosen.
	changed, err := sf.Update(SettingsUpdate{AutoInstall: &autoInstall})
	require.NoError(t, err)
	assert.True(t, changed)
	changed, err = sf.Update(SettingsUpdate{AutoInstall: &autoInstall})
	require.NoError(t, err)
	assert.False(t, changed)
	loaded, err := sf.Load()
	require.NoError(t, err)
	assert.Equal(t, "https://temporary.example/cjv", loaded.ResolveDistServer())
	var stored map[string]any
	_, err = toml.DecodeFile(sf.Path(), &stored)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"version": int64(1), "auto_install": true}, stored)
}

func TestSettingsFileFailedUpdateKeepsCache(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvFallbackSettings, filepath.Join(dir, "missing-fallback.toml"))
	sf := NewSettingsFile(filepath.Join(dir, "settings.toml"))
	before, err := sf.Load()
	require.NoError(t, err)
	// A directory at the destination prevents the atomic file replacement.
	require.NoError(t, os.Mkdir(sf.Path(), 0o700))
	autoInstall := false
	changed, err := sf.Update(SettingsUpdate{AutoInstall: &autoInstall})
	require.Error(t, err)
	assert.False(t, changed)
	after, err := sf.Load()
	require.NoError(t, err)
	assert.Equal(t, before, after)
}

func TestSettingsFileRejectsInvalidDocumentBeforePublishing(t *testing.T) {
	for _, operation := range []string{"save", "update"} {
		t.Run(operation, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv(EnvFallbackSettings, filepath.Join(dir, "missing-fallback.toml"))
			sf := NewSettingsFile(filepath.Join(dir, "settings.toml"))
			initial := DefaultSettings()
			initial.DefaultToolchain = "lts-1.0.5"
			require.NoError(t, sf.Save(&initial))
			before, err := os.ReadFile(sf.Path())
			require.NoError(t, err)
			// The TOML encoder permits these bytes, but a settings reader
			// rejects the document. Failure must leave the old reference intact.
			invalid := "sdk-\xff"
			if operation == "save" {
				settings, err := sf.Load()
				require.NoError(t, err)
				settings.DefaultToolchain = invalid
				require.Error(t, sf.Save(settings))
			} else {
				changed, err := sf.Update(SettingsUpdate{DefaultToolchain: &invalid})
				require.Error(t, err)
				assert.False(t, changed)
			}
			after, err := os.ReadFile(sf.Path())
			require.NoError(t, err)
			assert.Equal(t, before, after)
			cached, err := sf.Load()
			require.NoError(t, err)
			assert.Equal(t, initial.DefaultToolchain, cached.DefaultToolchain)
		})
	}
}
