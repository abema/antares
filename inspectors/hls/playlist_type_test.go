package hls

import (
	"testing"

	"github.com/abema/antares/core"
	"github.com/grafov/m3u8"
	"github.com/stretchr/testify/require"
)

func TestPlaylistTypeInspector(t *testing.T) {
	testCases := []struct {
		name                  string
		playlistTypeCondition PlaylistTypeCondition
		endlistCondition      EndlistCondition
		mediaPlaylists        map[string]*core.MediaPlaylist
		severity              core.Severity
	}{
		{
			name:                  "PlaylistTypeMustOmitted/EndlistMustNotExist/OK",
			playlistTypeCondition: PlaylistTypeMustOmitted,
			endlistCondition:      EndlistMustNotExist,
			mediaPlaylists: map[string]*core.MediaPlaylist{
				"0.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{MediaType: 0, Closed: false}},
				"1.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{MediaType: 0, Closed: false}},
			},
			severity: core.Info,
		},
		{
			name:                  "PlaylistTypeMustOmitted/EndlistMustNotExist/EndlistError",
			playlistTypeCondition: PlaylistTypeMustOmitted,
			endlistCondition:      EndlistMustNotExist,
			mediaPlaylists: map[string]*core.MediaPlaylist{
				"0.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{MediaType: 0, Closed: false}},
				"1.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{MediaType: 0, Closed: true}},
			},
			severity: core.Error,
		},
		{
			name:                  "PlaylistTypeMustOmitted/EndlistMustNotExist/PlaylistTypeError",
			playlistTypeCondition: PlaylistTypeMustOmitted,
			endlistCondition:      EndlistMustNotExist,
			mediaPlaylists: map[string]*core.MediaPlaylist{
				"0.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{MediaType: m3u8.EVENT, Closed: false}},
				"1.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{MediaType: 0, Closed: false}},
			},
			severity: core.Error,
		},
		{
			name:                  "PlaylistTypeMustVOD/EndlistMustExist/OK",
			playlistTypeCondition: PlaylistTypeMustVOD,
			endlistCondition:      EndlistMustExist,
			mediaPlaylists: map[string]*core.MediaPlaylist{
				"0.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{MediaType: m3u8.VOD, Closed: true}},
				"1.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{MediaType: m3u8.VOD, Closed: true}},
			},
			severity: core.Info,
		},
		{
			name:                  "PlaylistTypeMustVOD/EndlistMustExist/EndlistError",
			playlistTypeCondition: PlaylistTypeMustVOD,
			endlistCondition:      EndlistMustExist,
			mediaPlaylists: map[string]*core.MediaPlaylist{
				"0.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{MediaType: m3u8.VOD, Closed: true}},
				"1.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{MediaType: m3u8.VOD, Closed: false}},
			},
			severity: core.Error,
		},
		{
			name:                  "PlaylistTypeMustVOD/EndlistAny/OK",
			playlistTypeCondition: PlaylistTypeMustVOD,
			endlistCondition:      EndlistAny,
			mediaPlaylists: map[string]*core.MediaPlaylist{
				"0.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{MediaType: m3u8.VOD, Closed: true}},
				"1.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{MediaType: m3u8.VOD, Closed: false}},
			},
			severity: core.Info,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ins := NewPlaylistTypeInspector(&PlaylistTypeInspectorConfig{
				PlaylistTypeCondition: tc.playlistTypeCondition,
				EndlistCondition:      tc.endlistCondition,
			})
			report := ins.Inspect(&core.Playlists{MediaPlaylists: tc.mediaPlaylists}, nil)
			require.Equal(t, tc.severity, report.Severity)
		})
	}
}
