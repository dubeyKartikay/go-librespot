package spclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// These persisted operations are Spotify's private Pathfinder protocol. Keep
// the hashes together so protocol changes fail explicitly and are easy to update.
// Wire format reference: sonic-liberation/spotube-plugin-spotify, 47d0a105,
// .bruno/Spotify GQL (request/response examples; no implementation is copied).
var browseOperations = map[string]string{
	"libraryV3":                 "390c78e5b951029bad359785e69b07b536a509c581cbcd0aded5e5067f187455",
	"searchDesktop":             "d9f785900f0710b31c07818d617f4f7600c1e21217e80f5b043d1e78d74e6026",
	"getAlbum":                  "b9bfabef66ed756e5e13f68a942deb60bd4125ec1f1be8cc42769dc0259b4b10",
	"queryArtistDiscographyAll": "5e07d323febb57b4a56a42abbf781490e58764aa45feb6e3dc0591564fc56599",
}

type jsonObject = map[string]any

func (c *Spclient) browseQuery(ctx context.Context, operation string, variables jsonObject) (jsonObject, error) {
	body, err := json.Marshal(jsonObject{"operationName": operation, "variables": variables,
		"extensions": jsonObject{"persistedQuery": jsonObject{"version": 1, "sha256Hash": browseOperations[operation]}}})
	if err != nil {
		return nil, err
	}
	endpoint, _ := url.Parse("https://api-partner.spotify.com/pathfinder/v2/query")
	resp, err := c.innerRequest(ctx, http.MethodPost, endpoint, nil, http.Header{"Content-Type": {"application/json"}, "Accept": {"application/json"}}, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned HTTP %d", operation, resp.StatusCode)
	}
	var envelope struct {
		Data   jsonObject `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("decode %s: %w", operation, err)
	}
	if len(envelope.Errors) != 0 {
		return nil, fmt.Errorf("%s: %s", operation, envelope.Errors[0].Message)
	}
	if envelope.Data == nil {
		return nil, fmt.Errorf("%s: missing response data", operation)
	}
	return envelope.Data, nil
}

// Browse returns a Web-API-shaped page without using the public Web API. This
// keeps desktop consumers independent of Pathfinder's nested response schema.
func (c *Spclient) Browse(ctx context.Context, kind, uri, query string, offset, limit int) (jsonObject, error) {
	vars := jsonObject{"offset": offset, "limit": limit}
	var operation string
	var path []string
	switch kind {
	case "playlists", "albums", "artists":
		operation = "libraryV3"
		vars["filters"] = []string{strings.ToUpper(kind[:1]) + kind[1:]}
		vars["order"], vars["textFilter"], vars["folderUri"] = nil, "", nil
		vars["flatten"], vars["includeFoldersWhenFlattening"] = true, false
		vars["features"], vars["expandedFolders"] = []string{}, []string{}
		path = []string{"me", "libraryV3"}
	case "search-playlists", "search-tracks", "search-albums", "search-artists":
		operation = "searchDesktop"
		vars["searchTerm"], vars["numberOfTopResults"] = query, 5
		vars["includeAudiobooks"], vars["includeArtistHasConcertsField"] = true, false
		vars["includePreReleases"], vars["includeLocalConcertsField"], vars["includeAuthors"] = true, false, false
		field := strings.TrimPrefix(kind, "search-")
		if field == "tracks" {
			field = "tracksV2"
		}
		if field == "albums" {
			field = "albumsV2"
		}
		path = []string{"searchV2", field}
	case "artist-albums":
		operation = "queryArtistDiscographyAll"
		vars["uri"], vars["order"] = uri, "DATE_DESC"
		path = []string{"artistUnion", "discography", "all"}
	case "album-tracks":
		operation = "getAlbum"
		vars["uri"], vars["locale"] = uri, ""
		path = []string{"albumUnion", "tracksV2"}
	default:
		return nil, fmt.Errorf("unsupported browse kind %q", kind)
	}
	data, err := c.browseQuery(ctx, operation, vars)
	if err != nil {
		return nil, err
	}
	page := objectAt(data, path...)
	if page == nil {
		return nil, fmt.Errorf("%s: missing page %s", operation, strings.Join(path, "."))
	}
	rawItems, ok := page["items"].([]any)
	if !ok {
		return nil, fmt.Errorf("%s: missing page items", operation)
	}
	items := make([]any, 0)
	for _, raw := range rawItems {
		item, _ := raw.(map[string]any)
		if kind == "artist-albums" {
			releases := arrayAt(item, "releases", "items")
			if len(releases) == 0 {
				continue
			}
			item, _ = releases[0].(map[string]any)
		} else {
			item = unwrapBrowseItem(item)
		}
		mapped := mapBrowseEntity(item)
		if mapped == nil {
			continue
		}
		if kind == "albums" {
			mapped = jsonObject{"album": mapped}
		}
		items = append(items, mapped)
	}
	total, ok := page["totalCount"].(float64)
	if !ok {
		return nil, fmt.Errorf("%s: missing totalCount", operation)
	}
	result := jsonObject{"items": items, "total": total, "offset": offset, "limit": limit}
	if kind == "artists" {
		after := ""
		if len(rawItems) > 0 && offset+len(rawItems) < int(total) {
			after = fmt.Sprint(offset + len(rawItems))
		}
		result["cursors"] = jsonObject{"after": after}
	}
	return result, nil
}

func objectAt(obj jsonObject, keys ...string) jsonObject {
	for _, key := range keys {
		obj, _ = obj[key].(map[string]any)
		if obj == nil {
			return nil
		}
	}
	return obj
}

func arrayAt(obj jsonObject, keys ...string) []any {
	if len(keys) == 0 {
		return nil
	}
	parent := objectAt(obj, keys[:len(keys)-1]...)
	items, _ := parent[keys[len(keys)-1]].([]any)
	return items
}

func stringAt(obj jsonObject, keys ...string) string {
	parent := objectAt(obj, keys[:len(keys)-1]...)
	s, _ := parent[keys[len(keys)-1]].(string)
	return s
}

func unwrapBrowseItem(item jsonObject) jsonObject {
	for _, key := range []string{"item", "track", "data"} {
		if inner := objectAt(item, key); inner != nil {
			return unwrapBrowseItem(inner)
		}
	}
	return item
}

func mapBrowseEntity(item jsonObject) jsonObject {
	uri := stringAt(item, "uri")
	parts := strings.Split(uri, ":")
	if len(parts) != 3 {
		return nil
	}
	kind := parts[1]
	if kind != "track" && kind != "album" && kind != "artist" && kind != "playlist" {
		return nil
	}
	name := stringAt(item, "name")
	if name == "" {
		name = stringAt(item, "profile", "name")
	}
	result := jsonObject{"id": parts[2], "uri": uri, "type": kind, "name": name}
	images := arrayAt(item, "coverArt", "sources")
	if images == nil {
		images = arrayAt(item, "visuals", "avatarImage", "sources")
	}
	if images == nil {
		if covers := arrayAt(item, "images", "items"); len(covers) > 0 {
			cover, _ := covers[0].(map[string]any)
			images = arrayAt(cover, "sources")
		}
	}
	if images == nil {
		images = []any{}
	}
	result["images"] = images
	artists := make([]any, 0)
	for _, raw := range arrayAt(item, "artists", "items") {
		artist, _ := raw.(map[string]any)
		if mapped := mapBrowseEntity(unwrapBrowseItem(artist)); mapped != nil {
			artists = append(artists, mapped)
		}
	}
	result["artists"] = artists
	result["description"] = stringAt(item, "description")
	if kind == "track" {
		result["duration_ms"] = objectAt(item, "duration")["totalMilliseconds"]
		if album := objectAt(item, "albumOfTrack"); album != nil {
			result["album"] = mapBrowseEntity(album)
		}
	}
	if kind == "album" {
		result["album_type"] = strings.ToLower(stringAt(item, "type"))
	}
	return result
}
