package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
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
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sf.Load()
		}()
	}
	wg.Wait()
}
