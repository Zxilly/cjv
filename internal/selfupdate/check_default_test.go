//go:build !mirror

package selfupdate

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCheckFetchesMetadataWithoutInstalling(t *testing.T) {
	url := serveUpdateRelease(t, "v2.0.0", nil, "")
	for _, test := range []struct {
		current string
		status  Status
		latest  string
	}{{"1.0.0", StatusAvailable, "2.0.0"}, {"2.0.0", StatusUpToDate, "2.0.0"}, {"dev", StatusDevelopment, "dev"}} {
		result, err := Check(t.Context(), url, test.current)
		require.NoError(t, err)
		require.Equal(t, test.status, result.Status)
		require.Equal(t, test.latest, result.Version)
	}
}
