package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type browseRequest struct {
	Kind, URI, Query string
	Offset, Limit    int
}

func (s *ConcreteApiServer) registerBrowseRoutes(m *http.ServeMux) {
	m.HandleFunc("/browse/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		data, err := parseBrowseRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.handleRequest(ApiRequest{Type: "browse", Data: data}, w)
	})
}

func parseBrowseRequest(r *http.Request) (browseRequest, error) {
	q := r.URL.Query()
	d := browseRequest{Kind: strings.TrimPrefix(r.URL.Path, "/browse/"), URI: q.Get("uri"), Query: q.Get("q"), Limit: 10}
	if d.Kind == "album-tracks" {
		d.Limit = 50
	}
	for key, target := range map[string]*int{"offset": &d.Offset, "limit": &d.Limit} {
		if raw := q.Get(key); raw != "" {
			value, err := strconv.Atoi(raw)
			if err != nil || value < 0 || (key == "limit" && (value == 0 || value > 50)) {
				return d, fmt.Errorf("invalid %s", key)
			}
			*target = value
		}
	}
	switch d.Kind {
	case "user", "first-track", "playlists", "tracks", "albums", "artists":
	case "artist-albums", "album-tracks":
		kind := "album"
		if d.Kind == "artist-albums" {
			kind = "artist"
		}
		parts := strings.Split(d.URI, ":")
		if len(parts) != 3 || parts[0] != "spotify" || parts[1] != kind || parts[2] == "" {
			return d, fmt.Errorf("invalid %s URI", kind)
		}
	case "search-playlists", "search-tracks", "search-albums", "search-artists":
		if strings.TrimSpace(d.Query) == "" {
			return d, fmt.Errorf("query is required")
		}
	default:
		return d, fmt.Errorf("unknown browse endpoint")
	}
	return d, nil
}

func (p *AppPlayer) browse(ctx context.Context, d browseRequest) (any, error) {
	if d.Kind == "user" {
		return map[string]string{"id": p.sess.Username()}, nil
	}
	if d.Kind == "tracks" || d.Kind == "first-track" {
		if d.Kind == "first-track" {
			d.Offset, d.Limit = 0, 1
		}
		// Liked songs are a Spotify context, resolved using the same protocol as playlists.
		uri := "spotify:user:" + p.sess.Username() + ":collection"
		resolved, err := p.resolveTracks(ctx, ApiRequestDataResolveTracks{Uri: uri, Offset: d.Offset, Limit: d.Limit})
		if err != nil {
			return nil, err
		}
		if d.Kind == "first-track" {
			if len(resolved.Tracks) == 0 {
				return nil, ErrNotFound
			}
			return map[string]string{"uri": resolved.Tracks[0].Uri}, nil
		}
		items := make([]any, 0, len(resolved.Tracks))
		for _, track := range resolved.Tracks {
			artists := make([]any, 0, len(track.Artists))
			for _, name := range track.Artists {
				artists = append(artists, map[string]string{"name": name})
			}
			items = append(items, map[string]any{"track": map[string]any{
				"uri": track.Uri, "name": track.Name, "artists": artists, "duration_ms": track.DurationMs,
				"album": map[string]any{"uri": track.AlbumUri, "name": track.AlbumName, "images": []any{map[string]string{"url": track.Img}}},
			}})
		}
		return map[string]any{"items": items, "offset": resolved.Offset, "limit": resolved.Limit, "total": resolved.Total}, nil
	}
	return p.sess.Spclient().Browse(ctx, d.Kind, d.URI, d.Query, d.Offset, d.Limit)
}
