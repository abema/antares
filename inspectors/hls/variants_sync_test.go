package hls

import (
	"testing"

	"github.com/abema/antares/v2/core"
	m3u8 "github.com/abema/go-simple-m3u8"
	"github.com/stretchr/testify/require"
)

func TestVariantsSyncInspector(t *testing.T) {
	segments := func(begin, end int, dur float64) []*m3u8.Segment {
		segments := make([]*m3u8.Segment, 0)
		for i := begin; i < end; i++ {
			segment := &m3u8.Segment{
				Tags:     m3u8.SegmentTags{},
				Sequence: int64(i),
			}
			segment.Tags.SetExtInfValue(dur, 64)
			segments = append(segments, segment)
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
