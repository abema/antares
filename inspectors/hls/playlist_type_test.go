package hls

import (
	"testing"

	"github.com/abema/antares/v2/core"
	m3u8 "github.com/abema/go-simple-m3u8"
	"github.com/stretchr/testify/require"
)

func TestPlaylistTypeInspector(t *testing.T) {
	eventTypeTags := m3u8.MediaPlaylistTags{}
	eventTypeTags.SetPlaylistType(m3u8.MediaPlaylistTypeEvent)
	vodTypeTags := m3u8.MediaPlaylistTags{}
	vodTypeTags.SetPlaylistType(m3u8.MediaPlaylistTypeVOD)

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
				"0.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{EndList: false}},
				"1.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{EndList: false}},
			},
			severity: core.Info,
		},
		{
			name:                  "PlaylistTypeMustOmitted/EndlistMustNotExist/EndlistError",
			playlistTypeCondition: PlaylistTypeMustOmitted,
			endlistCondition:      EndlistMustNotExist,
			mediaPlaylists: map[string]*core.MediaPlaylist{
				"0.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{EndList: false}},
				"1.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{EndList: true}},
			},
			severity: core.Error,
		},
		{
			name:                  "PlaylistTypeMustOmitted/EndlistMustNotExist/PlaylistTypeError",
			playlistTypeCondition: PlaylistTypeMustOmitted,
			endlistCondition:      EndlistMustNotExist,
			mediaPlaylists: map[string]*core.MediaPlaylist{
				"0.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{Tags: eventTypeTags, EndList: false}},
				"1.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{EndList: false}},
			},
			severity: core.Error,
		},
		{
			name:                  "PlaylistTypeMustVOD/EndlistMustExist/OK",
			playlistTypeCondition: PlaylistTypeMustVOD,
			endlistCondition:      EndlistMustExist,
			mediaPlaylists: map[string]*core.MediaPlaylist{
				"0.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{Tags: vodTypeTags, EndList: true}},
				"1.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{Tags: vodTypeTags, EndList: true}},
			},
			severity: core.Info,
		},
		{
			name:                  "PlaylistTypeMustVOD/EndlistMustExist/EndlistError",
			playlistTypeCondition: PlaylistTypeMustVOD,
			endlistCondition:      EndlistMustExist,
			mediaPlaylists: map[string]*core.MediaPlaylist{
				"0.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{Tags: vodTypeTags, EndList: true}},
				"1.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{Tags: vodTypeTags, EndList: false}},
			},
			severity: core.Error,
		},
		{
			name:                  "PlaylistTypeMustVOD/EndlistAny/OK",
			playlistTypeCondition: PlaylistTypeMustVOD,
			endlistCondition:      EndlistAny,
			mediaPlaylists: map[string]*core.MediaPlaylist{
				"0.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{Tags: vodTypeTags, EndList: true}},
				"1.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{Tags: vodTypeTags, EndList: false}},
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
