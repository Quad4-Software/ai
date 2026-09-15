// SPDX-License-Identifier: 0BSD
// Command melovian exposes a running Melovian instance to MCP
// clients: library search and stats, playlist and smart-playlist
// management, metadata lookups and autofix, extension inspection and
// settings, plus optional SearXNG web search.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"sync"

	"github.com/Quad4-Software/ai/mcp/melovian/internal/mcp"
	"github.com/Quad4-Software/ai/mcp/melovian/internal/melovian"
)

var (
	clientOnce sync.Once
	client     *melovian.Client
	clientErr  error
)

// getClient lazily builds the shared client so tools/list works even
// when MELOVIAN_URL is unset; the first real call reports the error.
func getClient() (*melovian.Client, error) {
	clientOnce.Do(func() {
		client, clientErr = melovian.NewClient(melovian.ConfigFromEnv())
	})
	return client, clientErr
}

// strArg, intArg, boolArg, objArg, obj build input schemas compactly.
func strArg(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func intArg(desc string) map[string]any {
	return map[string]any{"type": "integer", "description": desc}
}

func boolArg(desc string) map[string]any {
	return map[string]any{"type": "boolean", "description": desc}
}

func objArg(desc string) map[string]any {
	return map[string]any{"type": "object", "description": desc}
}

func obj(props map[string]any, req ...string) map[string]any {
	return map[string]any{"type": "object", "properties": props, "required": req}
}

// idRe bounds IDs embedded in URL paths. Melovian IDs are slugs and
// hex hashes; anything else is rejected rather than escaped.
var idRe = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_-]{0,127}$`)

func validID(name, v string) error {
	if !idRe.MatchString(v) {
		return fmt.Errorf("invalid %s %q; want [A-Za-z0-9_-] up to 128 chars", name, v)
	}
	return nil
}

// render marshals a decoded value to indented JSON for the agent.
func render(v any) (string, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode result: %w", err)
	}
	return string(b), nil
}

// jsonArgs decodes the raw arguments object.
func jsonArgs(args json.RawMessage, dst any) error {
	if len(args) == 0 {
		return fmt.Errorf("missing arguments")
	}
	return json.Unmarshal(args, dst)
}

func tools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "status",
			Description: "Probe the Melovian instance: health, auth state, config, and whether the configured credentials authenticate.",
			InputSchema: obj(map[string]any{}),
			Handle: func(ctx context.Context, _ json.RawMessage) (string, error) {
				c, err := getClient()
				if err != nil {
					return "", err
				}
				out, err := c.Status(ctx)
				if err != nil {
					return "", err
				}
				return render(out)
			},
		},
		{
			Name:        "library_stats",
			Description: "Aggregate library counts: artists, albums, tracks, and related totals.",
			InputSchema: obj(map[string]any{}),
			Handle: func(ctx context.Context, _ json.RawMessage) (string, error) {
				c, err := getClient()
				if err != nil {
					return "", err
				}
				out, err := c.LibraryStats(ctx)
				if err != nil {
					return "", err
				}
				return render(out)
			},
		},
		{
			Name:        "search",
			Description: "Search the library over artists, albums, and songs. Returns matching entities, not a flat track list.",
			InputSchema: obj(map[string]any{
				"q":     strArg("search text, matched across artist names, album titles, and song titles"),
				"limit": intArg("max results per entity type; server default applies when 0"),
			}, "q"),
			InputExamples: []map[string]any{
				{"arguments": json.RawMessage(`{"q": "aphex", "limit": 10}`)},
			},
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Q     string `json:"q"`
					Limit int    `json:"limit"`
				}
				if err := jsonArgs(args, &a); err != nil {
					return "", err
				}
				c, err := getClient()
				if err != nil {
					return "", err
				}
				out, err := c.Search(ctx, a.Q, a.Limit)
				if err != nil {
					return "", err
				}
				return render(out)
			},
		},
		{
			Name:        "artists",
			Description: "List library artists with album counts. Large libraries return many entries; prefer search when looking for a known name.",
			InputSchema: obj(map[string]any{}),
			Handle: func(ctx context.Context, _ json.RawMessage) (string, error) {
				c, err := getClient()
				if err != nil {
					return "", err
				}
				out, err := c.SimpleGet(ctx, "/api/local-music/artists")
				if err != nil {
					return "", err
				}
				return render(out)
			},
		},
		{
			Name:        "albums",
			Description: "List library albums. Large libraries return many entries; prefer search when looking for a known title.",
			InputSchema: obj(map[string]any{}),
			Handle: func(ctx context.Context, _ json.RawMessage) (string, error) {
				c, err := getClient()
				if err != nil {
					return "", err
				}
				out, err := c.SimpleGet(ctx, "/api/local-music/albums")
				if err != nil {
					return "", err
				}
				return render(out)
			},
		},
		{
			Name:        "album_tracks",
			Description: "List the tracks on one album.",
			InputSchema: obj(map[string]any{
				"albumId": strArg("album ID from search or albums"),
			}, "albumId"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					AlbumID string `json:"albumId"`
				}
				if err := jsonArgs(args, &a); err != nil {
					return "", err
				}
				if err := validID("albumId", a.AlbumID); err != nil {
					return "", err
				}
				c, err := getClient()
				if err != nil {
					return "", err
				}
				out, err := c.SimpleGet(ctx, "/api/local-music/albums/"+a.AlbumID)
				if err != nil {
					return "", err
				}
				return render(out)
			},
		},
		{
			Name:        "genres",
			Description: "List genres present in the library.",
			InputSchema: obj(map[string]any{}),
			Handle: func(ctx context.Context, _ json.RawMessage) (string, error) {
				c, err := getClient()
				if err != nil {
					return "", err
				}
				out, err := c.SimpleGet(ctx, "/api/local-music/genres")
				if err != nil {
					return "", err
				}
				return render(out)
			},
		},
		{
			Name:        "genre_tracks",
			Description: "List tracks in one genre. Use genres to get valid names.",
			InputSchema: obj(map[string]any{
				"genre": strArg("genre name, for example shoegaze"),
				"count": intArg("max tracks; server default applies when 0"),
			}, "genre"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Genre string `json:"genre"`
					Count int    `json:"count"`
				}
				if err := jsonArgs(args, &a); err != nil {
					return "", err
				}
				c, err := getClient()
				if err != nil {
					return "", err
				}
				out, err := c.GenreTracks(ctx, a.Genre, a.Count)
				if err != nil {
					return "", err
				}
				return render(out)
			},
		},
		{
			Name:        "random_tracks",
			Description: "Random library tracks; a starting point for mixes when the agent has no other signal.",
			InputSchema: obj(map[string]any{
				"size": intArg("number of tracks, for example 20"),
			}),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Size int `json:"size"`
				}
				_ = json.Unmarshal(args, &a)
				c, err := getClient()
				if err != nil {
					return "", err
				}
				out, err := c.RandomTracks(ctx, a.Size)
				if err != nil {
					return "", err
				}
				return render(out)
			},
		},
		{
			Name:        "track",
			Description: "Fetch one track by ID: title, artist, album, duration, play count, genre.",
			InputSchema: obj(map[string]any{
				"trackId": strArg("track ID from search or a track list"),
			}, "trackId"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					TrackID string `json:"trackId"`
				}
				if err := jsonArgs(args, &a); err != nil {
					return "", err
				}
				if err := validID("trackId", a.TrackID); err != nil {
					return "", err
				}
				c, err := getClient()
				if err != nil {
					return "", err
				}
				s, err := c.Song(ctx, a.TrackID)
				if err != nil {
					return "", err
				}
				return render(s)
			},
		},
		{
			Name:        "similar_tracks",
			Description: "Tracks similar to a given track; the primary building block for generating mixes.",
			InputSchema: obj(map[string]any{
				"trackId": strArg("seed track ID"),
			}, "trackId"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					TrackID string `json:"trackId"`
				}
				if err := jsonArgs(args, &a); err != nil {
					return "", err
				}
				if err := validID("trackId", a.TrackID); err != nil {
					return "", err
				}
				c, err := getClient()
				if err != nil {
					return "", err
				}
				out, err := c.SimpleGet(ctx, "/api/local-music/songs/"+a.TrackID+"/similar")
				if err != nil {
					return "", err
				}
				return render(out)
			},
		},
		{
			Name:        "starred",
			Description: "Starred/favorited tracks; useful as taste signal for mixes.",
			InputSchema: obj(map[string]any{}),
			Handle: func(ctx context.Context, _ json.RawMessage) (string, error) {
				c, err := getClient()
				if err != nil {
					return "", err
				}
				out, err := c.SimpleGet(ctx, "/api/local-music/starred")
				if err != nil {
					return "", err
				}
				return render(out)
			},
		},
		{
			Name:        "playlists",
			Description: "List playlists (metadata only, no track contents). Use playlist to fetch one playlist's tracks.",
			InputSchema: obj(map[string]any{}),
			Handle: func(ctx context.Context, _ json.RawMessage) (string, error) {
				c, err := getClient()
				if err != nil {
					return "", err
				}
				out, err := c.SimpleGet(ctx, "/api/music/playlists")
				if err != nil {
					return "", err
				}
				return render(out)
			},
		},
		{
			Name:        "playlist",
			Description: "Fetch one playlist with its full track list.",
			InputSchema: obj(map[string]any{
				"playlistId": strArg("playlist ID from playlists"),
			}, "playlistId"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					PlaylistID string `json:"playlistId"`
				}
				if err := jsonArgs(args, &a); err != nil {
					return "", err
				}
				if err := validID("playlistId", a.PlaylistID); err != nil {
					return "", err
				}
				c, err := getClient()
				if err != nil {
					return "", err
				}
				out, err := c.SimpleGet(ctx, "/api/music/playlists/"+a.PlaylistID)
				if err != nil {
					return "", err
				}
				return render(out)
			},
		},
		{
			Name:        "playlist_create",
			Description: "Create a new empty playlist. Add tracks with playlist_set_tracks or playlist_add_track.",
			Write:       true,
			InputSchema: obj(map[string]any{
				"name": strArg("playlist name"),
			}, "name"),
			InputExamples: []map[string]any{
				{"arguments": json.RawMessage(`{"name": "Rainy day mix"}`)},
			},
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Name string `json:"name"`
				}
				if err := jsonArgs(args, &a); err != nil {
					return "", err
				}
				c, err := getClient()
				if err != nil {
					return "", err
				}
				out, err := c.CreatePlaylist(ctx, a.Name)
				if err != nil {
					return "", err
				}
				return render(out)
			},
		},
		{
			Name:        "playlist_rename",
			Description: "Rename a playlist.",
			Write:       true,
			InputSchema: obj(map[string]any{
				"playlistId": strArg("playlist ID"),
				"name":       strArg("new name"),
			}, "playlistId", "name"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					PlaylistID string `json:"playlistId"`
					Name       string `json:"name"`
				}
				if err := jsonArgs(args, &a); err != nil {
					return "", err
				}
				if err := validID("playlistId", a.PlaylistID); err != nil {
					return "", err
				}
				c, err := getClient()
				if err != nil {
					return "", err
				}
				var out any
				if err := c.Put(ctx, "/api/music/playlists/"+a.PlaylistID, map[string]string{"name": a.Name}, &out); err != nil {
					return "", err
				}
				return render(out)
			},
		},
		{
			Name:        "playlist_delete",
			Description: "Delete a playlist permanently.",
			Write:       true,
			InputSchema: obj(map[string]any{
				"playlistId": strArg("playlist ID"),
			}, "playlistId"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					PlaylistID string `json:"playlistId"`
				}
				if err := jsonArgs(args, &a); err != nil {
					return "", err
				}
				if err := validID("playlistId", a.PlaylistID); err != nil {
					return "", err
				}
				c, err := getClient()
				if err != nil {
					return "", err
				}
				var out any
				if err := c.Delete(ctx, "/api/music/playlists/"+a.PlaylistID, &out); err != nil {
					return "", err
				}
				return render(out)
			},
		},
		{
			Name:        "playlist_set_tracks",
			Description: "Replace a playlist's contents with an ordered list of track IDs. Track metadata is resolved automatically.",
			Write:       true,
			InputSchema: obj(map[string]any{
				"playlistId": strArg("playlist ID"),
				"trackIds":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "ordered track IDs"},
			}, "playlistId", "trackIds"),
			InputExamples: []map[string]any{
				{"arguments": json.RawMessage(`{"playlistId": "pl-1", "trackIds": ["tr-a", "tr-b"]}`)},
			},
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					PlaylistID string   `json:"playlistId"`
					TrackIDs   []string `json:"trackIds"`
				}
				if err := jsonArgs(args, &a); err != nil {
					return "", err
				}
				if err := validID("playlistId", a.PlaylistID); err != nil {
					return "", err
				}
				for _, id := range a.TrackIDs {
					if err := validID("trackId", id); err != nil {
						return "", err
					}
				}
				if len(a.TrackIDs) > 2000 {
					return "", fmt.Errorf("trackIds capped at 2000; got %d", len(a.TrackIDs))
				}
				c, err := getClient()
				if err != nil {
					return "", err
				}
				out, err := c.SetPlaylistTracks(ctx, a.PlaylistID, a.TrackIDs)
				if err != nil {
					return "", err
				}
				return render(out)
			},
		},
		{
			Name:        "playlist_add_track",
			Description: "Append one track to a playlist.",
			Write:       true,
			InputSchema: obj(map[string]any{
				"playlistId": strArg("playlist ID"),
				"trackId":    strArg("track ID to append"),
			}, "playlistId", "trackId"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					PlaylistID string `json:"playlistId"`
					TrackID    string `json:"trackId"`
				}
				if err := jsonArgs(args, &a); err != nil {
					return "", err
				}
				if err := validID("playlistId", a.PlaylistID); err != nil {
					return "", err
				}
				if err := validID("trackId", a.TrackID); err != nil {
					return "", err
				}
				c, err := getClient()
				if err != nil {
					return "", err
				}
				out, err := c.AddPlaylistTrack(ctx, a.PlaylistID, a.TrackID)
				if err != nil {
					return "", err
				}
				return render(out)
			},
		},
		{
			Name:        "playlist_remove_track",
			Description: "Remove one track from a playlist.",
			Write:       true,
			InputSchema: obj(map[string]any{
				"playlistId": strArg("playlist ID"),
				"trackId":    strArg("track ID to remove"),
			}, "playlistId", "trackId"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					PlaylistID string `json:"playlistId"`
					TrackID    string `json:"trackId"`
				}
				if err := jsonArgs(args, &a); err != nil {
					return "", err
				}
				if err := validID("playlistId", a.PlaylistID); err != nil {
					return "", err
				}
				if err := validID("trackId", a.TrackID); err != nil {
					return "", err
				}
				c, err := getClient()
				if err != nil {
					return "", err
				}
				out, err := c.RemovePlaylistTrack(ctx, a.PlaylistID, a.TrackID)
				if err != nil {
					return "", err
				}
				return render(out)
			},
		},
		{
			Name:        "smart_playlist_support",
			Description: "Check whether smart playlists are supported by the active library backend before building rules.",
			InputSchema: obj(map[string]any{}),
			Handle: func(ctx context.Context, _ json.RawMessage) (string, error) {
				c, err := getClient()
				if err != nil {
					return "", err
				}
				out, err := c.SimpleGet(ctx, "/api/music/smart-playlists/support")
				if err != nil {
					return "", err
				}
				return render(out)
			},
		},
		{
			Name:        "smart_playlist_preview",
			Description: "Preview which tracks a smart playlist ruleset would select, without saving it.",
			InputSchema: obj(map[string]any{
				"rules": objArg("rules object, for example {\"all\":[{\"field\":\"genre\",\"op\":\"is\",\"value\":\"ambient\"}]}"),
				"limit": intArg("max preview rows"),
			}, "rules"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Rules map[string]any `json:"rules"`
					Limit int            `json:"limit"`
				}
				if err := jsonArgs(args, &a); err != nil {
					return "", err
				}
				c, err := getClient()
				if err != nil {
					return "", err
				}
				out, err := c.PreviewSmartPlaylist(ctx, a.Rules, a.Limit)
				if err != nil {
					return "", err
				}
				return render(out)
			},
		},
		{
			Name:        "smart_playlist_create",
			Description: "Create a saved smart playlist from a rules object. Preview first with smart_playlist_preview.",
			Write:       true,
			InputSchema: obj(map[string]any{
				"name":    strArg("playlist name"),
				"rules":   objArg("rules object, same shape as smart_playlist_preview"),
				"comment": strArg("optional description"),
				"public":  boolArg("share publicly on a Subsonic-compatible server"),
			}, "name", "rules"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Name    string         `json:"name"`
					Rules   map[string]any `json:"rules"`
					Comment string         `json:"comment"`
					Public  bool           `json:"public"`
				}
				if err := jsonArgs(args, &a); err != nil {
					return "", err
				}
				c, err := getClient()
				if err != nil {
					return "", err
				}
				out, err := c.CreateSmartPlaylist(ctx, a.Name, a.Rules, a.Comment, a.Public)
				if err != nil {
					return "", err
				}
				return render(out)
			},
		},
		{
			Name:        "metadata_summary",
			Description: "Counts of tracks with metadata issues: unknown artist or album, missing title.",
			InputSchema: obj(map[string]any{}),
			Handle: func(ctx context.Context, _ json.RawMessage) (string, error) {
				c, err := getClient()
				if err != nil {
					return "", err
				}
				out, err := c.SimpleGet(ctx, "/api/local-music/metadata/summary")
				if err != nil {
					return "", err
				}
				return render(out)
			},
		},
		{
			Name:        "metadata_tracks",
			Description: "List tracks with their metadata issue flags.",
			InputSchema: obj(map[string]any{
				"q": strArg("optional filter text"),
			}),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Q string `json:"q"`
				}
				_ = json.Unmarshal(args, &a)
				c, err := getClient()
				if err != nil {
					return "", err
				}
				v := url.Values{}
				if a.Q != "" {
					v.Set("q", a.Q)
				}
				var out any
				if err := c.Get(ctx, "/api/local-music/metadata/tracks", v, &out); err != nil {
					return "", err
				}
				return render(out)
			},
		},
		{
			Name:        "metadata_lookup",
			Description: "Look up enhanced metadata for a track or a free-form query: artist, title, album, MusicBrainz matches.",
			InputSchema: obj(map[string]any{
				"trackId": strArg("optional library track ID; supplies artist/title/album automatically"),
				"q":       strArg("free-form lookup text"),
				"artist":  strArg("artist name"),
				"title":   strArg("track title"),
				"album":   strArg("album title"),
				"source":  strArg("lookup source, for example musicbrainz"),
				"limit":   intArg("max matches"),
			}),
			InputExamples: []map[string]any{
				{"arguments": json.RawMessage(`{"artist": "Boards of Canada", "title": "Dayvan Cowboy"}`)},
			},
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					TrackID string `json:"trackId"`
					Q       string `json:"q"`
					Artist  string `json:"artist"`
					Title   string `json:"title"`
					Album   string `json:"album"`
					Source  string `json:"source"`
					Limit   int    `json:"limit"`
				}
				if err := jsonArgs(args, &a); err != nil {
					return "", err
				}
				v := url.Values{}
				for k, val := range map[string]string{
					"trackId": a.TrackID, "q": a.Q, "artist": a.Artist,
					"title": a.Title, "album": a.Album, "source": a.Source,
				} {
					if val != "" {
						v.Set(k, val)
					}
				}
				if a.Limit > 0 {
					v.Set("limit", strconv.Itoa(a.Limit))
				}
				c, err := getClient()
				if err != nil {
					return "", err
				}
				out, err := c.MetadataLookup(ctx, v)
				if err != nil {
					return "", err
				}
				return render(out)
			},
		},
		{
			Name:        "metadata_suggestions",
			Description: "Suggested metadata corrections for one track.",
			InputSchema: obj(map[string]any{
				"trackId": strArg("library track ID"),
			}, "trackId"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					TrackID string `json:"trackId"`
				}
				if err := jsonArgs(args, &a); err != nil {
					return "", err
				}
				if err := validID("trackId", a.TrackID); err != nil {
					return "", err
				}
				c, err := getClient()
				if err != nil {
					return "", err
				}
				out, err := c.SimpleGet(ctx, "/api/local-music/metadata/tracks/"+a.TrackID+"/suggestions")
				if err != nil {
					return "", err
				}
				return render(out)
			},
		},
		{
			Name:        "metadata_autofix",
			Description: "Apply automatic metadata corrections to one track. Mutates library metadata.",
			Write:       true,
			InputSchema: obj(map[string]any{
				"trackId": strArg("library track ID"),
			}, "trackId"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					TrackID string `json:"trackId"`
				}
				if err := jsonArgs(args, &a); err != nil {
					return "", err
				}
				if err := validID("trackId", a.TrackID); err != nil {
					return "", err
				}
				c, err := getClient()
				if err != nil {
					return "", err
				}
				var out any
				if err := c.Post(ctx, "/api/local-music/metadata/tracks/"+a.TrackID+"/autofix", map[string]any{}, &out); err != nil {
					return "", err
				}
				return render(out)
			},
		},
		{
			Name:        "extensions",
			Description: "List installed and bundled extensions with enabled state and settings.",
			InputSchema: obj(map[string]any{}),
			Handle: func(ctx context.Context, _ json.RawMessage) (string, error) {
				c, err := getClient()
				if err != nil {
					return "", err
				}
				out, err := c.SimpleGet(ctx, "/api/extensions")
				if err != nil {
					return "", err
				}
				return render(out)
			},
		},
		{
			Name:        "extension_registry",
			Description: "List extensions available in the configured registry index.",
			InputSchema: obj(map[string]any{}),
			Handle: func(ctx context.Context, _ json.RawMessage) (string, error) {
				c, err := getClient()
				if err != nil {
					return "", err
				}
				out, err := c.SimpleGet(ctx, "/api/extensions/registry")
				if err != nil {
					return "", err
				}
				return render(out)
			},
		},
		{
			Name:        "extension_settings",
			Description: "Read the saved settings for one installed extension.",
			InputSchema: obj(map[string]any{
				"id": strArg("extension ID"),
			}, "id"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					ID string `json:"id"`
				}
				if err := jsonArgs(args, &a); err != nil {
					return "", err
				}
				if err := validID("id", a.ID); err != nil {
					return "", err
				}
				c, err := getClient()
				if err != nil {
					return "", err
				}
				out, err := c.SimpleGet(ctx, "/api/extensions/"+a.ID+"/settings")
				if err != nil {
					return "", err
				}
				return render(out)
			},
		},
		{
			Name:        "extension_set_settings",
			Description: "Replace an extension's settings object, for example to apply a color palette or toggle a visual option.",
			Write:       true,
			InputSchema: obj(map[string]any{
				"id":       strArg("extension ID"),
				"settings": objArg("settings object; keys must match the extension's declared settings schema"),
			}, "id", "settings"),
			InputExamples: []map[string]any{
				{"arguments": json.RawMessage(`{"id": "visualizations", "settings": {"palette": "ember", "particles": "stars"}}`)},
			},
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					ID       string         `json:"id"`
					Settings map[string]any `json:"settings"`
				}
				if err := jsonArgs(args, &a); err != nil {
					return "", err
				}
				if err := validID("id", a.ID); err != nil {
					return "", err
				}
				if a.Settings == nil {
					return "", fmt.Errorf("missing required argument: settings")
				}
				c, err := getClient()
				if err != nil {
					return "", err
				}
				var out any
				if err := c.Put(ctx, "/api/extensions/"+a.ID+"/settings", map[string]any{"settings": a.Settings}, &out); err != nil {
					return "", err
				}
				return render(out)
			},
		},
		{
			Name:        "extension_enable",
			Description: "Enable or disable an installed extension, for example to apply a theme extension's styling.",
			Write:       true,
			InputSchema: obj(map[string]any{
				"id":      strArg("extension ID"),
				"enabled": boolArg("true to enable, false to disable"),
			}, "id", "enabled"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					ID      string `json:"id"`
					Enabled bool   `json:"enabled"`
				}
				if err := jsonArgs(args, &a); err != nil {
					return "", err
				}
				if err := validID("id", a.ID); err != nil {
					return "", err
				}
				c, err := getClient()
				if err != nil {
					return "", err
				}
				var out any
				if err := c.Put(ctx, "/api/extensions/"+a.ID+"/enabled", map[string]bool{"enabled": a.Enabled}, &out); err != nil {
					return "", err
				}
				return render(out)
			},
		},
		{
			Name:        "extension_install_remote",
			Description: "Install an extension from the trusted registry index by ID. The package URL comes from the index, not the request.",
			Write:       true,
			InputSchema: obj(map[string]any{
				"id":      strArg("extension ID in the registry index"),
				"version": strArg("optional version; latest when empty"),
			}, "id"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					ID      string `json:"id"`
					Version string `json:"version"`
				}
				if err := jsonArgs(args, &a); err != nil {
					return "", err
				}
				if err := validID("id", a.ID); err != nil {
					return "", err
				}
				c, err := getClient()
				if err != nil {
					return "", err
				}
				var out any
				if err := c.Post(ctx, "/api/extensions/install-remote", map[string]string{"id": a.ID, "version": a.Version}, &out); err != nil {
					return "", err
				}
				return render(out)
			},
		},
		{
			Name:        "web_search",
			Description: "Search the web through a configured SearXNG instance (MELOVIAN_SEARXNG_URL). Useful for music metadata research. Disabled when not configured.",
			InputSchema: obj(map[string]any{
				"q":     strArg("search query"),
				"limit": intArg("max results, default 10"),
			}, "q"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Q     string `json:"q"`
					Limit int    `json:"limit"`
				}
				if err := jsonArgs(args, &a); err != nil {
					return "", err
				}
				c, err := getClient()
				if err != nil {
					return "", err
				}
				out, err := c.WebSearch(ctx, a.Q, a.Limit)
				if err != nil {
					return "", err
				}
				return render(out)
			},
		},
		{
			Name:        "scaffold_extension",
			Description: "Generate source files for a new Melovian registry extension: manifest, script.ts, CSS, CHANGELOG. Returns file contents; nothing is written to disk.",
			InputSchema: obj(map[string]any{
				"id":          strArg("extension slug, [a-z0-9-], for example mood-lights"),
				"name":        strArg("display name; defaults to id"),
				"description": strArg("one-line description"),
				"author":      strArg("author name"),
			}, "id"),
			InputExamples: []map[string]any{
				{"arguments": json.RawMessage(`{"id": "mood-lights", "name": "Mood Lights", "author": "Melovian"}`)},
			},
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					ID          string `json:"id"`
					Name        string `json:"name"`
					Description string `json:"description"`
					Author      string `json:"author"`
				}
				if err := jsonArgs(args, &a); err != nil {
					return "", err
				}
				files, err := melovian.ScaffoldExtension(a.ID, a.Name, a.Description, a.Author)
				if err != nil {
					return "", err
				}
				return render(map[string]any{"files": files})
			},
		},
	}
}

func main() {
	srv := mcp.NewServer("melovian", "0.1.0", tools(), nil)
	if err := srv.Serve(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "melovian:", err)
		os.Exit(1)
	}
}
