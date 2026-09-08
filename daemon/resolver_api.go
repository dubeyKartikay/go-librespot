package daemon

import (
	"net/http"
	"strconv"
	"strings"
)

const ApiRequestTypeResolveTracks ApiRequestType = "resolve_tracks"

type ApiRequestDataResolveTracks struct {
	Uri    string `json:"uri"`
	Offset int    `json:"offset"`
	Limit  int    `json:"limit"`
}
type ApiResponseResolvedTrack struct {
	Uri        string            `json:"uri"`
	Uid        string            `json:"uid"`
	Name       string            `json:"name"`
	Artists    []string          `json:"artists"`
	Img        string            `json:"img"`
	AlbumName  string            `json:"album_name"`
	DurationMs int               `json:"duration_ms"`
	AlbumUri   string            `json:"album_uri"`
	ArtistUri  string            `json:"artist_uri"`
	Metadata   map[string]string `json:"metadata"`
}

type ApiResponseResolveTracks struct {
	Uri     string                     `json:"uri"`
	Offset  int                        `json:"offset"`
	Limit   int                        `json:"limit"`
	Total   int                        `json:"total"`
	HasNext bool                       `json:"has_next"`
	Tracks  []ApiResponseResolvedTrack `json:"tracks"`
}

func (s *ConcreteApiServer) registerResolverRoutes(m *http.ServeMux) {
	m.HandleFunc("/resolver/tracks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		query := r.URL.Query()
		uri := strings.TrimSpace(query.Get("uri"))
		if len(uri) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		offset := 0
		if rawOffset := strings.TrimSpace(query.Get("offset")); len(rawOffset) > 0 {
			parsed, err := strconv.Atoi(rawOffset)
			if err != nil || parsed < 0 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			offset = parsed
		}

		limit := 50
		if rawLimit := strings.TrimSpace(query.Get("limit")); len(rawLimit) > 0 {
			parsed, err := strconv.Atoi(rawLimit)
			if err != nil || parsed <= 0 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			limit = parsed
		}
		if limit > 200 {
			limit = 200
		}

		s.handleRequest(ApiRequest{Type: ApiRequestTypeResolveTracks, Data: ApiRequestDataResolveTracks{Uri: uri, Offset: offset, Limit: limit}}, w)
	})

}
