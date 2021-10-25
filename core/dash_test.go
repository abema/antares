package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zencoder/go-dash/helpers/ptrs"
	"github.com/zencoder/go-dash/mpd"
)

func TestSegments(t *testing.T) {
	t.Run("shared_SegmentTemplate", func(t *testing.T) {
		m := Manifest{
			URL: "https://localhost/foo/manifest.mpd",
			MPD: &mpd.MPD{
				BaseURL: "./bar/",
				Periods: []*mpd.Period{
					{
						AdaptationSets: []*mpd.AdaptationSet{
							{
								SegmentTemplate: &mpd.SegmentTemplate{
									Initialization: ptrs.Strptr("$RepresentationID$/init.mp4"),
									Media:          ptrs.Strptr("$RepresentationID$/$Time$.mp4"),
									SegmentTimeline: &mpd.SegmentTimeline{
										Segments: []*mpd.SegmentTimelineSegment{
											{StartTime: ptrs.Uint64ptr(1000000), Duration: 90000},
											{Duration: 80000, RepeatCount: ptrs.Intptr(2)},
											{Duration: 70000, RepeatCount: ptrs.Intptr(1)},
										},
									},
								},
								Representations: []*mpd.Representation{
									{ID: ptrs.Strptr("r0")},
									{ID: ptrs.Strptr("r1")},
								},
							},
						},
					},
				},
			},
		}
		segments, err := m.Segments()
		require.NoError(t, err)
		require.Len(t, segments, 14)

		assert.Equal(t, "https://localhost/foo/bar/r0/init.mp4", segments[0].URL)
		assert.True(t, segments[0].Initialization)

		assert.Equal(t, "https://localhost/foo/bar/r0/1000000.mp4", segments[1].URL)
		assert.False(t, segments[1].Initialization)
		assert.Equal(t, uint64(1000000), segments[1].Time)
		assert.Equal(t, uint64(90000), segments[1].Duration)

		assert.Equal(t, "https://localhost/foo/bar/r0/1090000.mp4", segments[2].URL)
		assert.Equal(t, "https://localhost/foo/bar/r0/1170000.mp4", segments[3].URL)
		assert.Equal(t, "https://localhost/foo/bar/r0/1250000.mp4", segments[4].URL)
		assert.Equal(t, "https://localhost/foo/bar/r0/1330000.mp4", segments[5].URL)
		assert.Equal(t, "https://localhost/foo/bar/r0/1400000.mp4", segments[6].URL)
		assert.Equal(t, "https://localhost/foo/bar/r1/init.mp4", segments[7].URL)
		assert.Equal(t, "https://localhost/foo/bar/r1/1000000.mp4", segments[8].URL)
		assert.Equal(t, "https://localhost/foo/bar/r1/1090000.mp4", segments[9].URL)
		assert.Equal(t, "https://localhost/foo/bar/r1/1170000.mp4", segments[10].URL)
		assert.Equal(t, "https://localhost/foo/bar/r1/1250000.mp4", segments[11].URL)
		assert.Equal(t, "https://localhost/foo/bar/r1/1330000.mp4", segments[12].URL)
		assert.Equal(t, "https://localhost/foo/bar/r1/1400000.mp4", segments[13].URL)
	})

	t.Run("individual_SegmentTemplate", func(t *testing.T) {
		m := Manifest{
			URL: "https://localhost/foo/manifest.mpd",
			MPD: &mpd.MPD{
				BaseURL: "./bar/",
				Periods: []*mpd.Period{
					{
						AdaptationSets: []*mpd.AdaptationSet{
							{
								Representations: []*mpd.Representation{
									{
										ID: ptrs.Strptr("r0"),
										SegmentTemplate: &mpd.SegmentTemplate{
											Initialization: ptrs.Strptr("$RepresentationID$/init.mp4"),
											Media:          ptrs.Strptr("$RepresentationID$/$Time$.mp4"),
											SegmentTimeline: &mpd.SegmentTimeline{
												Segments: []*mpd.SegmentTimelineSegment{
													{StartTime: ptrs.Uint64ptr(1000000), Duration: 90000, RepeatCount: ptrs.Intptr(2)},
												},
											},
										},
									},
									{
										ID: ptrs.Strptr("r1"),
										SegmentTemplate: &mpd.SegmentTemplate{
											Initialization: ptrs.Strptr("$RepresentationID$/init.mp4"),
											Media:          ptrs.Strptr("$RepresentationID$/$Time$.mp4"),
											SegmentTimeline: &mpd.SegmentTimeline{
												Segments: []*mpd.SegmentTimelineSegment{
													{StartTime: ptrs.Uint64ptr(1000000), Duration: 90000, RepeatCount: ptrs.Intptr(2)},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}
		segments, err := m.Segments()
		require.NoError(t, err)
		require.Len(t, segments, 8)

		assert.Equal(t, "https://localhost/foo/bar/r0/init.mp4", segments[0].URL)
		assert.True(t, segments[0].Initialization)

		assert.Equal(t, "https://localhost/foo/bar/r0/1000000.mp4", segments[1].URL)
		assert.False(t, segments[1].Initialization)
		assert.Equal(t, uint64(1000000), segments[1].Time)
		assert.Equal(t, uint64(90000), segments[1].Duration)

		assert.Equal(t, "https://localhost/foo/bar/r1/init.mp4", segments[4].URL)
		assert.True(t, segments[4].Initialization)

		assert.Equal(t, "https://localhost/foo/bar/r1/1000000.mp4", segments[5].URL)
		assert.False(t, segments[5].Initialization)
		assert.Equal(t, uint64(1000000), segments[5].Time)
		assert.Equal(t, uint64(90000), segments[5].Duration)
	})
}

func TestDASHManifestDownloader(t *testing.T) {
	manifest := []byte(`<MPD type="dynamic" minimumUpdatePeriod="PT5.000000S" availabilityStartTime="1970-01-01T00:00:00Z">` +
		`<Period id="1" start="PT1600000000.000S">` +
		`<AdaptationSet mimeType="video/mp4">` +
		`<SegmentTemplate timescale="90000" media="media_$RepresentationID$_$Time$.mp4" initialization="media_$RepresentationID$_init.mp4">` +
		`<SegmentTimeline>` +
		`<S d="900000" t="0"/><S d="900000"/>` +
		`</SegmentTimeline>` +
		`</SegmentTemplate>` +
		`<Representation id="4M" bandwidth="4000000" codecs="avc1.4D401F" frameRate="30000/1001" height="1080" width="1920" />` +
		`<Representation id="2M" bandwidth="2000000" codecs="avc1.4D401F" frameRate="30000/1001" height="720" width="1280" />` +
		`</AdaptationSet>` +
		`</Period>` +
		`</MPD>`)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/manifest.mpd":
			w.Write(manifest)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	d := newDASHManifestDownloader(newClient(http.DefaultClient, nil, nil), time.Second)
	mpd, err := d.Download(context.Background(), server.URL+"/manifest.mpd")
	require.NoError(t, err)
	require.Equal(t, server.URL+"/manifest.mpd", mpd.URL)
	require.Len(t, mpd.Raw, len(manifest))
	require.Equal(t, "dynamic", *mpd.Type)
	require.Len(t, mpd.Periods, 1)
}
