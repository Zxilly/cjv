//go:build openharmony

package ohos

import (
	"context"
	"debug/elf"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/ohos/selfsign"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testELF(kind elf.Type) []byte {
	// ELF64 with a null section and a section-name string table.
	names := []byte("\x00.shstrtab\x00")
	b := make([]byte, 64+128+len(names))
	copy(b, []byte{0x7f, 'E', 'L', 'F', 2, 1, 1})
	binary.LittleEndian.PutUint16(b[16:], uint16(kind))
	binary.LittleEndian.PutUint16(b[18:], uint16(elf.EM_AARCH64))
	binary.LittleEndian.PutUint32(b[20:], 1)
	binary.LittleEndian.PutUint64(b[40:], 64)
	binary.LittleEndian.PutUint16(b[52:], 64)
	binary.LittleEndian.PutUint16(b[58:], 64)
	binary.LittleEndian.PutUint16(b[60:], 2)
	binary.LittleEndian.PutUint16(b[62:], 1)
	binary.LittleEndian.PutUint32(b[128:], 1)
	binary.LittleEndian.PutUint32(b[132:], uint32(elf.SHT_STRTAB))
	binary.LittleEndian.PutUint64(b[152:], 192)
	binary.LittleEndian.PutUint64(b[160:], uint64(len(names)))
	copy(b[192:], names)
	return b
}

func TestPrepareNativePayloads(t *testing.T) {
	root := t.TempDir()
	signed, err := selfsign.Sign(testELF(elf.ET_DYN))
	require.NoError(t, err)
	files := map[string][]byte{
		"cjc": testELF(elf.ET_EXEC), "lib.so": testELF(elf.ET_DYN),
		"object.o": testELF(elf.ET_REL), "signed.so": signed, "readme": []byte("plain text"),
	}
	for name, data := range files {
		require.NoError(t, os.WriteFile(filepath.Join(root, name), data, 0o755))
	}
	require.NoError(t, PrepareTree(context.Background(), root))
	for name, original := range files {
		got, err := os.ReadFile(filepath.Join(root, name))
		require.NoError(t, err)
		if name == "cjc" || name == "lib.so" {
			require.NoError(t, selfsign.Verify(got))
			got[9] ^= 1 // Input bytes, outside the signature payload.
			assert.Error(t, selfsign.Verify(got))
		} else {
			assert.Equal(t, original, got, name)
		}
	}
}

func TestSigningFailurePreservesInput(t *testing.T) {
	for _, invalidOutput := range []bool{false, true} {
		root := t.TempDir()
		path := filepath.Join(root, "cjc")
		original := testELF(elf.ET_EXEC)
		require.NoError(t, os.WriteFile(path, original, 0o755))
		err := prepareTree(context.Background(), root, func(_ context.Context, _, output string) error {
			if invalidOutput {
				return os.WriteFile(output, []byte("not an ELF"), 0o600)
			}
			return errors.New("sign failed")
		})
		require.Error(t, err)
		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, original, got)
		entries, err := os.ReadDir(root)
		require.NoError(t, err)
		assert.Len(t, entries, 1)
	}
}

func TestPreparationDoesNotFollowExternalSymlinks(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside")
	original := testELF(elf.ET_EXEC)
	require.NoError(t, os.WriteFile(outside, original, 0o755))
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	require.NoError(t, PrepareTree(context.Background(), root))
	got, err := os.ReadFile(outside)
	require.NoError(t, err)
	assert.Equal(t, original, got)
}

func TestSignerRejectsMalformedSectionOffsets(t *testing.T) {
	b := testELF(elf.ET_EXEC)
	binary.LittleEndian.PutUint64(b[152:], ^uint64(0)-1)
	_, err := selfsign.Sign(b)
	require.Error(t, err)
}
