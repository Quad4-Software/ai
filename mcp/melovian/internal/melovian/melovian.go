// SPDX-License-Identifier: 0BSD
package melovian

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Song is the local library track view returned by /api/local-music.
type Song struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Album     string `json:"album"`
	AlbumID   string `json:"albumId"`
	Artist    string `json:"artist"`
	ArtistID  string `json:"artistId"`
	Track     int    `json:"track"`
	Duration  int    `json:"duration"`
	Genre     string `json:"genre,omitempty"`
	PlayCount int    `json:"playCount,omitempty"`
}

// PlaylistTrack is the wire shape the playlist track endpoints accept.
// Field names must stay camelCase: the server decodes them into a
// struct without tags via case-insensitive matching.
type PlaylistTrack struct {
	TrackID    string `json:"trackId"`
	TrackTitle string `json:"trackTitle"`
	ArtistName string `json:"artistName"`
	AlbumID    string `json:"albumId"`
	AlbumTitle string `json:"albumTitle"`
	DurationMs int    `json:"durationMs"`
	CoverArtID string `json:"coverArtId"`
	Position   int    `json:"position,omitempty"`
}

// Status is a connectivity probe over the public endpoints.
func (c *Client) Status(ctx context.Context) (map[string]any, error) {
	var health map[string]any
	_ = c.Get(ctx, "/health", nil, &health)
	var auth map[string]any
	_ = c.Get(ctx, "/api/auth/status", nil, &auth)
	var cfg map[string]any
	_ = c.Get(ctx, "/api/config", nil, &cfg)
	out := map[string]any{"health": health, "auth": auth, "config": cfg}
	if err := c.Get(ctx, "/api/music/library-stats", nil, &map[string]any{}); err != nil {
		out["authenticated"] = false
		out["authenticatedError"] = err.Error()
	} else {
		out["authenticated"] = true
	}
	return out, nil
}

// Get fetches one endpoint into out. Use for simple read tools.
func (c *Client) getJSON(ctx context.Context, p string, q url.Values) (any, error) {
	var out any
	if err := c.Get(ctx, p, q, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// LibraryStats returns the aggregate counts the app reports.
func (c *Client) LibraryStats(ctx context.Context) (any, error) {
	return c.getJSON(ctx, "/api/music/library-stats", nil)
}

// Search runs a library-wide search over artists, albums, and songs.
func (c *Client) Search(ctx context.Context, q string, limit int) (any, error) {
	if q == "" {
		return nil, fmt.Errorf("missing required argument: q")
	}
	v := url.Values{"q": {q}}
	if limit > 0 {
		v.Set("limit", strconv.Itoa(limit))
	}
	return c.getJSON(ctx, "/api/local-music/search", v)
}

// SimpleGet wraps a plain GET with no parameters.
func (c *Client) SimpleGet(ctx context.Context, p string) (any, error) {
	return c.getJSON(ctx, p, nil)
}

// GenreTracks returns tracks for one genre.
func (c *Client) GenreTracks(ctx context.Context, genre string, count int) (any, error) {
	if genre == "" {
		return nil, fmt.Errorf("missing required argument: genre")
	}
	v := url.Values{"genre": {genre}}
	if count > 0 {
		v.Set("count", strconv.Itoa(count))
	}
	return c.getJSON(ctx, "/api/local-music/songsByGenre", v)
}

// RandomTracks returns size random library tracks.
func (c *Client) RandomTracks(ctx context.Context, size int) (any, error) {
	v := url.Values{}
	if size > 0 {
		v.Set("size", strconv.Itoa(size))
	}
	return c.getJSON(ctx, "/api/local-music/randomSongs", v)
}

// Song fetches one track view by ID.
func (c *Client) Song(ctx context.Context, id string) (Song, error) {
	var s Song
	if id == "" {
		return s, fmt.Errorf("missing required argument: trackId")
	}
	return s, c.Get(ctx, "/api/local-music/songs/"+id, nil, &s)
}

// playlistTrack resolves a track ID to the full wire shape the
// playlist endpoints store.
func (c *Client) playlistTrack(ctx context.Context, trackID string) (PlaylistTrack, error) {
	s, err := c.Song(ctx, trackID)
	if err != nil {
		return PlaylistTrack{}, fmt.Errorf("track %q: %w", trackID, err)
	}
	return PlaylistTrack{
		TrackID:    s.ID,
		TrackTitle: s.Title,
		ArtistName: s.Artist,
		AlbumID:    s.AlbumID,
		AlbumTitle: s.Album,
		DurationMs: s.Duration,
	}, nil
}

// PlaylistTracks resolves a list of track IDs in order.
func (c *Client) PlaylistTracks(ctx context.Context, ids []string) ([]PlaylistTrack, error) {
	tracks := make([]PlaylistTrack, 0, len(ids))
	for _, id := range ids {
		t, err := c.playlistTrack(ctx, id)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, t)
	}
	return tracks, nil
}

// CreatePlaylist creates an empty playlist and returns its view.
func (c *Client) CreatePlaylist(ctx context.Context, name string) (any, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("missing required argument: name")
	}
	var out any
	return out, c.Post(ctx, "/api/music/playlists",
		map[string]string{"name": name}, &out)
}

// SetPlaylistTracks replaces a playlist's track list with the given
// track IDs, resolved to full track records server-side first.
func (c *Client) SetPlaylistTracks(ctx context.Context, playlistID string, ids []string) (any, error) {
	if playlistID == "" {
		return nil, fmt.Errorf("missing required argument: playlistId")
	}
	tracks, err := c.PlaylistTracks(ctx, ids)
	if err != nil {
		return nil, err
	}
	var out any
	return out, c.Put(ctx, "/api/music/playlists/"+playlistID+"/tracks",
		map[string]any{"tracks": tracks}, &out)
}

// AddPlaylistTrack appends one track ID to a playlist.
func (c *Client) AddPlaylistTrack(ctx context.Context, playlistID, trackID string) (any, error) {
	if playlistID == "" {
		return nil, fmt.Errorf("missing required argument: playlistId")
	}
	t, err := c.playlistTrack(ctx, trackID)
	if err != nil {
		return nil, err
	}
	var out any
	return out, c.Post(ctx, "/api/music/playlists/"+playlistID+"/tracks", t, &out)
}

// RemovePlaylistTrack removes one track from a playlist.
func (c *Client) RemovePlaylistTrack(ctx context.Context, playlistID, trackID string) (any, error) {
	if playlistID == "" || trackID == "" {
		return nil, fmt.Errorf("missing required argument: playlistId and trackId")
	}
	var out any
	return out, c.Delete(ctx,
		"/api/music/playlists/"+playlistID+"/tracks/"+trackID, &out)
}

// PreviewSmartPlaylist runs rules without saving. The server wants the
// rules object as a JSON string under rulesJson.
func (c *Client) PreviewSmartPlaylist(ctx context.Context, rules map[string]any, limit int) (any, error) {
	if len(rules) == 0 {
		return nil, fmt.Errorf("missing required argument: rules")
	}
	rj, err := json.Marshal(rules)
	if err != nil {
		return nil, fmt.Errorf("encode rules: %w", err)
	}
	var out any
	body := map[string]any{"rulesJson": string(rj), "limit": limit}
	return out, c.Post(ctx, "/api/music/smart-playlists/preview", body, &out)
}

// CreateSmartPlaylist saves a smart playlist.
func (c *Client) CreateSmartPlaylist(ctx context.Context, name string, rules map[string]any, comment string, public bool) (any, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("missing required argument: name")
	}
	if len(rules) == 0 {
		return nil, fmt.Errorf("missing required argument: rules")
	}
	var out any
	return out, c.Post(ctx, "/api/music/smart-playlists", map[string]any{
		"name": name, "rules": rules, "comment": comment, "public": public,
	}, &out)
}

// MetadataLookup proxies the metadata lookup endpoint.
func (c *Client) MetadataLookup(ctx context.Context, q url.Values) (any, error) {
	return c.getJSON(ctx, "/api/local-music/metadata/lookup", q)
}
