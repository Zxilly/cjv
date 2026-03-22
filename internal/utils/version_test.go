package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersion_ReturnsNonEmpty(t *testing.T) {
	v := Version()
	assert.NotEmpty(t, v, "version should never be empty (defaults to 'dev')")
}
