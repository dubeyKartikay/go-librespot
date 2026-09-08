package daemon

import (
	"net/http/httptest"
	"testing"
)

func TestBrowseValidation(t *testing.T) {
	for _, path := range []string{
		"/browse/unknown", "/browse/tracks?offset=-1", "/browse/tracks?limit=0",
		"/browse/tracks?limit=51", "/browse/tracks?offset=invalid",
		"/browse/artist-albums?uri=spotify:album:abc", "/browse/album-tracks",
		"/browse/search-tracks?q=+",
	} {
		if _, err := parseBrowseRequest(httptest.NewRequest("GET", path, nil)); err == nil {
			t.Errorf("accepted %s", path)
		}
	}
	for _, path := range []string{
		"/browse/user", "/browse/first-track", "/browse/playlists", "/browse/tracks", "/browse/albums", "/browse/artists",
		"/browse/search-playlists?q=test", "/browse/search-tracks?q=test", "/browse/search-albums?q=test", "/browse/search-artists?q=test",
		"/browse/artist-albums?uri=spotify:artist:abc", "/browse/album-tracks?uri=spotify:album:abc",
	} {
		if _, err := parseBrowseRequest(httptest.NewRequest("GET", path, nil)); err != nil {
			t.Errorf("%s: %v", path, err)
		}
	}
}
