package selfupdate

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdateSkipsNetworkForUnsupportedOrDevBuilds(t *testing.T) {
	require.NoError(t, Update(context.Background(), "", "1.0.0"))
	require.NoError(t, Update(context.Background(), "https://example.com/owner/repo/releases", "dev"))
}
