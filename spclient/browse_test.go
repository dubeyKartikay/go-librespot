package spclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type browseTransport func(*http.Request) (*http.Response, error)

func (f browseTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestBrowsePrivateProtocolAndMapping(t *testing.T) {
	tests := []struct{ kind, operation, body, name string }{
		{"playlists", "libraryV3", `{"me":{"libraryV3":{"totalCount":11,"items":[{"item":{"data":{"uri":"spotify:playlist:p","name":"Mix","description":"desc","images":{"items":[{"sources":[{"url":"cover"}]}]}}}}]}}}`, "Mix"},
		{"albums", "libraryV3", `{"me":{"libraryV3":{"totalCount":11,"items":[{"item":{"data":{"uri":"spotify:album:a","name":"Album"}}}]}}}`, "Album"},
		{"artists", "libraryV3", `{"me":{"libraryV3":{"totalCount":11,"items":[{"item":{"data":{"uri":"spotify:artist:a","profile":{"name":"Artist"}}}}]}}}`, "Artist"},
		{"search-tracks", "searchDesktop", `{"searchV2":{"tracksV2":{"totalCount":11,"items":[{"item":{"data":{"uri":"spotify:track:t","name":"Song","artists":{"items":[{"uri":"spotify:artist:a","profile":{"name":"Artist"}}]},"albumOfTrack":{"uri":"spotify:album:a","name":"Album"},"duration":{"totalMilliseconds":1234}}}}]}}}`, "Song"},
		{"search-playlists", "searchDesktop", `{"searchV2":{"playlists":{"totalCount":11,"items":[{"data":{"uri":"spotify:playlist:p","name":"Mix"}}]}}}`, "Mix"},
		{"search-albums", "searchDesktop", `{"searchV2":{"albumsV2":{"totalCount":11,"items":[{"data":{"uri":"spotify:album:a","name":"Album"}}]}}}`, "Album"},
		{"search-artists", "searchDesktop", `{"searchV2":{"artists":{"totalCount":11,"items":[{"data":{"uri":"spotify:artist:a","profile":{"name":"Artist"}}}]}}}`, "Artist"},
		{"artist-albums", "queryArtistDiscographyAll", `{"artistUnion":{"discography":{"all":{"totalCount":11,"items":[{"releases":{"items":[{"uri":"spotify:album:a","name":"Album"}]}}]}}}}`, "Album"},
		{"album-tracks", "getAlbum", `{"albumUnion":{"tracksV2":{"totalCount":11,"items":[{"track":{"uri":"spotify:track:t","name":"Song"}}]}}}`, "Song"},
	}
	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			c := &Spclient{accessToken: func(context.Context, bool) (string, error) { return "session-token", nil },
				client: &http.Client{Transport: browseTransport(func(r *http.Request) (*http.Response, error) {
					if r.URL.String() != "https://api-partner.spotify.com/pathfinder/v2/query" || r.Method != "POST" {
						t.Errorf("unexpected URL: %s %s", r.Method, r.URL)
					}
					if r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Authorization") != "Bearer session-token" {
						t.Error("missing session auth or JSON content type")
					}
					var req map[string]any
					if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
						t.Fatal(err)
					}
					if req["operationName"] != tt.operation {
						t.Errorf("operation = %v", req["operationName"])
					}
					vars := objectAt(req, "variables")
					if vars["offset"] != float64(5) || vars["limit"] != float64(10) {
						t.Errorf("pagination = %v", vars)
					}
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"data":` + tt.body + `}`)), Header: http.Header{}}, nil
				})}}
			page, err := c.Browse(context.Background(), tt.kind, "spotify:album:a", "blue & train", 5, 10)
			if err != nil {
				t.Fatal(err)
			}
			items := page["items"].([]any)
			if len(items) != 1 {
				t.Fatalf("items = %#v", items)
			}
			item := items[0].(map[string]any)
			if tt.kind == "albums" {
				item = objectAt(item, "album")
			}
			if item["name"] != tt.name {
				t.Errorf("item = %#v", item)
			}
			if tt.kind == "artists" && stringAt(page, "cursors", "after") != "6" {
				t.Error("incorrect cursor")
			}
			if tt.kind == "search-tracks" {
				if item["duration_ms"] != float64(1234) || stringAt(item, "album", "name") != "Album" {
					t.Errorf("track = %#v", item)
				}
			}
		})
	}
}

func TestBrowseReportsProtocolFailures(t *testing.T) {
	for _, body := range []string{`{"errors":[{"message":"PersistedQueryNotFound"}]}`, `{"data":{}}`, `{"data":{"me":{"libraryV3":{"items":[]}}}}`, "{"} {
		c := &Spclient{accessToken: func(context.Context, bool) (string, error) { return "test", nil }, client: &http.Client{Transport: browseTransport(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
		})}}
		if _, err := c.Browse(context.Background(), "playlists", "", "", 0, 10); err == nil {
			t.Fatalf("accepted invalid response %s", body)
		}
	}
}
