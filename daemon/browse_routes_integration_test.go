//go:build test_integration

package daemon

import (
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestMergedBrowsingRoutes(t *testing.T) {
	ts := newTestServer(t, okReply)
	resp := ts.do(http.MethodGet, "/browse/playlists?offset=10&limit=10", nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	req := ts.request()
	require.Equal(t, ApiRequestType("browse"), req.Type)
	require.Equal(t, 10, req.Data.(browseRequest).Offset)
	resp = ts.do(http.MethodGet, "/resolver/tracks?uri=spotify:playlist:123&offset=20&limit=10", nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	req = ts.request()
	require.Equal(t, ApiRequestTypeResolveTracks, req.Type)
	require.Equal(t, 20, req.Data.(ApiRequestDataResolveTracks).Offset)
}
