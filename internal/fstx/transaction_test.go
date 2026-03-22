package fstx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTransaction_Commit_CleansBackups(t *testing.T) {
	dir := t.TempDir()
	prefix := filepath.Join(dir, "dest")
	os.MkdirAll(prefix, 0o755)

	// Create a file to be removed
	target := filepath.Join(prefix, "victim.txt")
	os.WriteFile(target, []byte("data"), 0o644)

	tx, err := NewTransaction(prefix)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	if err := tx.RemoveFile(target); err != nil {
		t.Fatal(err)
	}
	// File should be gone
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Error("file should be removed")
	}

	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	// Backup tmp dir should be cleaned
	if _, err := os.Stat(tx.tmpDir); !os.IsNotExist(err) {
		t.Error("tmp dir should be cleaned after commit")
	}
}

func TestTransaction_Rollback_RestoresRemovedFile(t *testing.T) {
	dir := t.TempDir()
	prefix := filepath.Join(dir, "dest")
	os.MkdirAll(prefix, 0o755)

	target := filepath.Join(prefix, "victim.txt")
	os.WriteFile(target, []byte("original"), 0o644)

	tx, err := NewTransaction(prefix)
	if err != nil {
		t.Fatal(err)
	}

	if err := tx.RemoveFile(target); err != nil {
		t.Fatal(err)
	}

	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal("file should be restored after rollback:", err)
	}
	if string(data) != "original" {
		t.Errorf("restored content = %q, want %q", string(data), "original")
	}
}

func TestTransaction_Rollback_RemovesAddedFile(t *testing.T) {
	dir := t.TempDir()
	prefix := filepath.Join(dir, "dest")
	os.MkdirAll(prefix, 0o755)

	src := filepath.Join(dir, "src.txt")
	os.WriteFile(src, []byte("new"), 0o644)
	dst := filepath.Join(prefix, "added.txt")

	tx, err := NewTransaction(prefix)
	if err != nil {
		t.Fatal(err)
	}

	if err := tx.AddFile(src, dst); err != nil {
		t.Fatal(err)
	}
	// File should exist
	if _, err := os.Stat(dst); err != nil {
		t.Fatal("added file should exist:", err)
	}

	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	// File should be gone after rollback
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Error("added file should be removed after rollback")
	}
}

func TestTransaction_Rollback_AfterCommit_IsNoop(t *testing.T) {
	dir := t.TempDir()
	tx, err := NewTransaction(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	// Should not panic or error
	if err := tx.Rollback(); err != nil {
		t.Fatal("rollback after commit should be noop:", err)
	}
}

func TestTransaction_RenameFile(t *testing.T) {
	dir := t.TempDir()
	prefix := filepath.Join(dir, "dest")
	os.MkdirAll(prefix, 0o755)

	from := filepath.Join(prefix, "old.txt")
	to := filepath.Join(prefix, "new.txt")
	os.WriteFile(from, []byte("content"), 0o644)

	tx, err := NewTransaction(prefix)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	if err := tx.RenameFile(from, to); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(from); !os.IsNotExist(err) {
		t.Error("old file should not exist")
	}
	data, _ := os.ReadFile(to)
	if string(data) != "content" {
		t.Error("new file should have content")
	}

	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}
