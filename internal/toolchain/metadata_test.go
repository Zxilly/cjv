package toolchain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNightlyReleaseMetadataRoundTrip(t *testing.T) {
	dir := t.TempDir()
	want := NightlyReleaseMetadata{
		ReleaseTag: "1.1.0-alpha.20260613020028",
		Version:    "1.2.0-alpha.20260613020028",
	}

	require.NoError(t, WriteNightlyReleaseMetadata(dir, want))
	got, err := ReadNightlyReleaseMetadata(dir)

	require.NoError(t, err)
	assert.Equal(t, want, got)
}
