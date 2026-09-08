# Browsing API

The daemon exposes read-only endpoints under `/browse/`, using its paired
session. They do not require a caller-supplied OAuth token.

| Endpoint suffix | Result / backing operation |
| --- | --- |
| user | Session username as `{"id":"…"}` |
| first-track | First liked track as `{"uri":"…"}`; 404 for an empty library |
| tracks | Liked songs through the context resolver |
| playlists, albums, artists | Pathfinder libraryV3 |
| search-playlists, search-tracks, search-albums, search-artists | Pathfinder searchDesktop |
| artist-albums | Pathfinder queryArtistDiscographyAll |
| album-tracks | Pathfinder getAlbum |

All routes require GET. Search requires `q`; artist-albums and album-tracks
require a Spotify `uri` of the corresponding type. Pages accept nonnegative
`offset` and `limit` between 1 and 50. Default limit is 10, or 50 for album
tracks. Results have `items`, `total`, `offset`, and `limit`, using Spotify
Web API compatible item shapes. Saved tracks/albums wrap their item in
`track`/`album`; artist pages additionally return `cursors.after` as the
next offset string, or an empty string at the end.

The existing `/resolver/tracks` endpoint remains the playlist-track API.
It and liked songs share context resolution and metadata enrichment. Partial
context downloads now return an error instead of claiming a complete total.

Unpaired sessions return 204, invalid parameters return 400, and unexpected
upstream/protocol failures return 500. No access tokens are returned by these
routes. Prefer a loopback server binding.

Pathfinder is a private protocol: persisted hashes can expire. Hashes and
response paths are centralized in `spclient/browse.go`. Protocol request
and response shapes were checked against the examples in
[spotube-plugin-spotify](https://github.com/sonic-liberation/spotube-plugin-spotify/tree/47d0a1051b576616f9e823cc756b84e8dc1a53f4/.bruno/Spotify%20GQL).
No client implementation was copied.

Use `credentials.type: device_auth` for pairing. `GET /auth/code` exposes the
pending URL, code, and expiry before a session exists. A 204 response means
there is no pending code. Credentials are saved using upstream's state store.
Rejected legacy login5 credentials trigger a fresh device authorization;
network failures still propagate without discarding stored credentials.
