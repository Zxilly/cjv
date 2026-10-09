//go:build openharmony

// Package ohos prepares downloaded native payloads for OpenHarmony execution.
package ohos

import (
	"context"
	"debug/elf"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/ohos/selfsign"
)

// PrepareTree signs unsigned ELF executables/shared libraries in private
// extraction scratch before publication or hardlink deduplication. It never
// follows symlinks or changes user-owned linked toolchains.
// Like Harmonybrew's rustup hook, ET_REL objects and existing signatures are
// left alone. Presence of .codesign is not cryptographic verification.
func PrepareTree(ctx context.Context, root string) error {
	return prepareTree(ctx, root, signBinary)
}

type signer func(context.Context, string, string) error

func prepareTree(ctx context.Context, root string, sign signer) error {
	info, err := os.Lstat(root)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("OpenHarmony preparation requires a private directory: %s", root)
	}
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		needed, err := needsSignature(path)
		if err != nil {
			return fmt.Errorf("inspect OpenHarmony payload %s: %w", path, err)
		}
		if !needed {
			return nil
		}
		return signReplacement(ctx, path, sign)
	})
}

func needsSignature(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close() //nolint:errcheck // read-only
	var magic [4]byte
	if _, err := io.ReadFull(f, magic[:]); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return false, nil
		}
		return false, err
	}
	if string(magic[:]) != "\x7fELF" {
		return false, nil
	}
	image, err := elf.NewFile(f)
	if err != nil {
		return false, err
	}
	return (image.Type == elf.ET_EXEC || image.Type == elf.ET_DYN) && image.Section(".codesign") == nil, nil
}

func signReplacement(ctx context.Context, path string, sign signer) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".cjv-sign-*")
	if err != nil {
		return err
	}
	output := f.Name()
	defer os.Remove(output) //nolint:errcheck // private scratch
	if err := f.Close(); err != nil {
		return err
	}
	if err := sign(ctx, path, output); err != nil {
		return fmt.Errorf("sign OpenHarmony payload %s: %w", path, err)
	}
	// Require a parseable signed ELF before replacing the extracted input.
	image, err := elf.Open(output)
	if err != nil {
		return fmt.Errorf("read signed ELF %s: %w", path, err)
	}
	signed := (image.Type == elf.ET_EXEC || image.Type == elf.ET_DYN) && image.Section(".codesign") != nil
	if err := image.Close(); err != nil {
		return err
	}
	if !signed {
		return fmt.Errorf("signer did not produce an ELF with a .codesign section: %s", path)
	}
	if err := os.Chmod(output, info.Mode().Perm()); err != nil {
		return err
	}
	// Destination and temporary file are siblings; failure leaves the input intact.
	return os.Rename(output, path)
}

func signBinary(ctx context.Context, input, output string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	raw, err := os.ReadFile(input)
	if err != nil {
		return err
	}
	signed, err := selfsign.Sign(raw)
	if err != nil {
		return err
	}
	if err := selfsign.Verify(signed); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return os.WriteFile(output, signed, 0o600)
}
