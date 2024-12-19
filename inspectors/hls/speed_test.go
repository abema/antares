package hls

import (
	"testing"
	"time"

	"github.com/abema/antares/v2/core"
	m3u8 "github.com/abema/go-simple-m3u8"
	"github.com/stretchr/testify/require"
)

func TestSpeedInspectorTest(t *testing.T) {
	segments := func(begin int) []*m3u8.Segment {
		segments := make([]*m3u8.Segment, 0)
		for i := begin; i < begin+10; i++ {
			segment := &m3u8.Segment{
				Tags:     m3u8.SegmentTags{},
				Sequence: int64(i),
			}
			segment.Tags.SetExtInfValue(10.0, 64)
			segments = append(segments, segment)
		}
		return segments
	}

	ins := NewSpeedInspectorWithConfig(&SpeedInspectorConfig{
		Interval: time.Minute,
		Warn:     15 * time.Second,
		Error:    30 * time.Second,
	})

	// first
	rep := ins.Inspect(&core.Playlists{
		MediaPlaylists: map[string]*core.MediaPlaylist{
			"0.m3u8": {
				URL:           "https://foo/0.m3u8",
				Time:          time.Unix(1000, 0),
				MediaPlaylist: &m3u8.MediaPlaylist{Segments: segments(10)},
			},
			"1.m3u8": {
				URL:           "https://foo/1.m3u8",
				Time:          time.Unix(1000, 0),
				MediaPlaylist: &m3u8.MediaPlaylist{Segments: segments(10)},
			},
		},
	}, nil)
	require.Equal(t, core.Info, rep.Severity)

	// 40 seconds later
	rep = ins.Inspect(&core.Playlists{
		MediaPlaylists: map[string]*core.MediaPlaylist{
			"0.m3u8": {
				URL:           "https://foo/0.m3u8",
				Time:          time.Unix(1040, 0),
				MediaPlaylist: &m3u8.MediaPlaylist{Segments: segments(14)},
			},
			"1.m3u8": {
				URL:           "https://foo/1.m3u8",
				Time:          time.Unix(1040, 0),
				MediaPlaylist: &m3u8.MediaPlaylist{Segments: segments(14)},
			},
		},
	}, nil)
	require.Equal(t, core.Info, rep.Severity)

	// 80 seconds later
	rep = ins.Inspect(&core.Playlists{
		MediaPlaylists: map[string]*core.MediaPlaylist{
			"0.m3u8": {
				URL:           "https://foo/0.m3u8",
				Time:          time.Unix(1080, 0),
				MediaPlaylist: &m3u8.MediaPlaylist{Segments: segments(18)},
			},
			"1.m3u8": {
				URL:           "https://foo/1.m3u8",
				Time:          time.Unix(1080, 0),
				MediaPlaylist: &m3u8.MediaPlaylist{Segments: segments(18)},
			},
		},
	}, nil)
	require.Equal(t, core.Info, rep.Severity)

	// 120 seconds later, low speed warning
	rep = ins.Inspect(&core.Playlists{
		MediaPlaylists: map[string]*core.MediaPlaylist{
			"0.m3u8": {
				URL:           "https://foo/0.m3u8",
				Time:          time.Unix(1120, 0),
				MediaPlaylist: &m3u8.MediaPlaylist{Segments: segments(22)},
			},
			"1.m3u8": {
				URL:           "https://foo/1.m3u8",
				Time:          time.Unix(1120, 0),
				MediaPlaylist: &m3u8.MediaPlaylist{Segments: segments(20)},
			},
		},
	}, nil)
	require.Equal(t, core.Warn, rep.Severity)

	// 160 seconds later, low speed error
	rep = ins.Inspect(&core.Playlists{
		MediaPlaylists: map[string]*core.MediaPlaylist{
			"0.m3u8": {
				URL:           "https://foo/0.m3u8",
				Time:          time.Unix(1160, 0),
				MediaPlaylist: &m3u8.MediaPlaylist{Segments: segments(26)},
			},
			"1.m3u8": {
				URL:           "https://foo/1.m3u8",
				Time:          time.Unix(1160, 0),
				MediaPlaylist: &m3u8.MediaPlaylist{Segments: segments(21)},
			},
		},
	}, nil)
	require.Equal(t, core.Error, rep.Severity)

	// 200 seconds later
	rep = ins.Inspect(&core.Playlists{
		MediaPlaylists: map[string]*core.MediaPlaylist{
			"0.m3u8": {
				URL:           "https://foo/0.m3u8",
				Time:          time.Unix(1200, 0),
				MediaPlaylist: &m3u8.MediaPlaylist{Segments: segments(30)},
			},
			"1.m3u8": {
				URL:           "https://foo/1.m3u8",
				Time:          time.Unix(1200, 0),
				MediaPlaylist: &m3u8.MediaPlaylist{Segments: segments(28)},
			},
		},
	}, nil)
	require.Equal(t, core.Info, rep.Severity)

	// 240 seconds later, high speed warning
	rep = ins.Inspect(&core.Playlists{
		MediaPlaylists: map[string]*core.MediaPlaylist{
			"0.m3u8": {
				URL:           "https://foo/0.m3u8",
				Time:          time.Unix(1240, 0),
				MediaPlaylist: &m3u8.MediaPlaylist{Segments: segments(36)},
			},
			"1.m3u8": {
				URL:           "https://foo/1.m3u8",
				Time:          time.Unix(1240, 0),
				MediaPlaylist: &m3u8.MediaPlaylist{Segments: segments(30)},
			},
		},
	}, nil)
	require.Equal(t, core.Warn, rep.Severity)

	// 280 seconds later, high speed warning
	rep = ins.Inspect(&core.Playlists{
		MediaPlaylists: map[string]*core.MediaPlaylist{
			"0.m3u8": {
				URL:           "https://foo/0.m3u8",
				Time:          time.Unix(1280, 0),
				MediaPlaylist: &m3u8.MediaPlaylist{Segments: segments(42)},
			},
			"1.m3u8": {
				URL:           "https://foo/1.m3u8",
				Time:          time.Unix(1280, 0),
				MediaPlaylist: &m3u8.MediaPlaylist{Segments: segments(36)},
			},
		},
	}, nil)
	require.Equal(t, core.Error, rep.Severity)

	// no segments
	rep = ins.Inspect(&core.Playlists{
		MediaPlaylists: map[string]*core.MediaPlaylist{
			"0.m3u8": {
				URL:           "https://foo/0.m3u8",
				Time:          time.Unix(1280, 0),
				MediaPlaylist: &m3u8.MediaPlaylist{Segments: nil},
			},
			"1.m3u8": {
				URL:           "https://foo/1.m3u8",
				Time:          time.Unix(1280, 0),
				MediaPlaylist: &m3u8.MediaPlaylist{Segments: nil},
			},
		},
	}, nil)
	require.Equal(t, core.Error, rep.Severity)

	// skip VOD variant
	rep = ins.Inspect(&core.Playlists{
		MediaPlaylists: map[string]*core.MediaPlaylist{
			"0.m3u8": {
				URL:           "https://foo/0.m3u8",
				Time:          time.Unix(1280, 0),
				MediaPlaylist: &m3u8.MediaPlaylist{Segments: nil, EndList: true},
			},
			"1.m3u8": {
				URL:           "https://foo/1.m3u8",
				Time:          time.Unix(1280, 0),
				MediaPlaylist: &m3u8.MediaPlaylist{Segments: nil},
			},
		},
	}, nil)
	require.Equal(t, core.Error, rep.Severity)

	// VOD only
	rep = ins.Inspect(&core.Playlists{
		MediaPlaylists: map[string]*core.MediaPlaylist{
			"0.m3u8": {
				URL:           "https://foo/0.m3u8",
				Time:          time.Unix(1280, 0),
				MediaPlaylist: &m3u8.MediaPlaylist{Segments: nil, EndList: true},
			},
			"1.m3u8": {
				URL:           "https://foo/1.m3u8",
				Time:          time.Unix(1280, 0),
				MediaPlaylist: &m3u8.MediaPlaylist{Segments: nil, EndList: true},
			},
		},
	}, nil)
	require.Equal(t, core.Info, rep.Severity)
}
