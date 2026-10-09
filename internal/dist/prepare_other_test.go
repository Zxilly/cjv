//go:build !openharmony

package dist

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOtherHostsDoNotInspectPayloads(t *testing.T) {
	require.NoError(t, preparePlatformTree(context.Background(), "nonexistent"))
}
