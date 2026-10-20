package dist

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPClientReportsAppVersionInUserAgent(t *testing.T) {
	old := AppVersion
	AppVersion = "1.2.3-test"
	t.Cleanup(func() { AppVersion = old })

	var got string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.UserAgent()
	}))
	defer server.Close()

	resp, err := newHTTPClient().Get(server.URL)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, "cjv/1.2.3-test", got)
}
