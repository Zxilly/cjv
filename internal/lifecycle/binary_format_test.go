package lifecycle

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestELFFormatDoesNotImplyLinux(t *testing.T) {
	format := formatFromMagic([]byte{0x7f, 'E', 'L', 'F'})
	assert.Equal(t, "ELF", format)
	assert.True(t, binaryFormatCompatible(format, "openharmony"))
	assert.True(t, binaryFormatCompatible(format, "linux"))
	assert.False(t, binaryFormatCompatible(format, "windows"))
	assert.False(t, binaryFormatCompatible("PE", "openharmony"))
	assert.False(t, binaryFormatCompatible("Mach-O", "openharmony"))
}
