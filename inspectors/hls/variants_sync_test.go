package hls

import (
	"testing"

	"github.com/abema/antares/core"
	"github.com/grafov/m3u8"
	"github.com/stretchr/testify/require"
)

func TestVariantsSyncInspector(t *testing.T) {
	segments := func(begin, end int, dur float64) []*m3u8.MediaSegment {
		segments := make([]*m3u8.MediaSegment, 0)
		for i := begin; i < end; i++ {
			segments = append(segments, &m3u8.MediaSegment{
				SeqId:    uint64(i),
				Duration: dur,
			})
		}
		return segments
	}

	t.Run("1-segment-late/ok", func(t *testing.T) {
		ins := NewVariantsSyncInspector()
		report := ins.Inspect(&core.Playlists{MediaPlaylists: map[string]*core.MediaPlaylist{
			"0.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{Segments: segments(10, 20, 10.0)}},
			"1.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{Segments: segments(10, 19, 10.0)}},
		}}, nil)
		require.Equal(t, core.Info, report.Severity)
	})

	t.Run("2-segments-late/warn", func(t *testing.T) {
		ins := NewVariantsSyncInspector()
		report := ins.Inspect(&core.Playlists{MediaPlaylists: map[string]*core.MediaPlaylist{
			"0.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{Segments: segments(10, 20, 10.0)}},
			"1.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{Segments: segments(10, 18, 10.0)}},
		}}, nil)
		require.Equal(t, core.Warn, report.Severity)
	})

	t.Run("3-segments-late/warn", func(t *testing.T) {
		ins := NewVariantsSyncInspector()
		report := ins.Inspect(&core.Playlists{MediaPlaylists: map[string]*core.MediaPlaylist{
			"0.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{Segments: segments(10, 20, 10.0)}},
			"1.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{Segments: segments(10, 17, 10.0)}},
		}}, nil)
		require.Equal(t, core.Warn, report.Severity)
	})

	t.Run("4-segments-late/error", func(t *testing.T) {
		ins := NewVariantsSyncInspector()
		report := ins.Inspect(&core.Playlists{MediaPlaylists: map[string]*core.MediaPlaylist{
			"0.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{Segments: segments(10, 20, 10.0)}},
			"1.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{Segments: segments(10, 16, 10.0)}},
		}}, nil)
		require.Equal(t, core.Error, report.Severity)
	})

	t.Run("400ms-difference/ok", func(t *testing.T) {
		ins := NewVariantsSyncInspector()
		report := ins.Inspect(&core.Playlists{MediaPlaylists: map[string]*core.MediaPlaylist{
			"0.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{Segments: segments(10, 20, 9.6)}},
			"1.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{Segments: segments(10, 20, 10.0)}},
		}}, nil)
		require.Equal(t, core.Info, report.Severity)
	})

	t.Run("500ms-difference/warn", func(t *testing.T) {
		ins := NewVariantsSyncInspector()
		report := ins.Inspect(&core.Playlists{MediaPlaylists: map[string]*core.MediaPlaylist{
			"0.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{Segments: segments(10, 20, 9.5)}},
			"1.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{Segments: segments(10, 20, 10.0)}},
		}}, nil)
		require.Equal(t, core.Warn, report.Severity)
	})

	t.Run("900ms-difference/warn", func(t *testing.T) {
		ins := NewVariantsSyncInspector()
		report := ins.Inspect(&core.Playlists{MediaPlaylists: map[string]*core.MediaPlaylist{
			"0.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{Segments: segments(10, 20, 9.1)}},
			"1.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{Segments: segments(10, 20, 10.0)}},
		}}, nil)
		require.Equal(t, core.Warn, report.Severity)
	})

	t.Run("1000ms-difference/warn", func(t *testing.T) {
		ins := NewVariantsSyncInspector()
		report := ins.Inspect(&core.Playlists{MediaPlaylists: map[string]*core.MediaPlaylist{
			"0.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{Segments: segments(10, 20, 9.0)}},
			"1.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{Segments: segments(10, 20, 10.0)}},
		}}, nil)
		require.Equal(t, core.Error, report.Severity)
	})
}
