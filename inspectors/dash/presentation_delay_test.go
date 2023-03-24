package dash

import (
	"testing"
	"time"

	"github.com/abema/antares/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zencoder/go-dash/helpers/ptrs"
	"github.com/zencoder/go-dash/mpd"
)

func TestPresentationDelayInspector(t *testing.T) {
	ins := NewPresentationDelayInspector()

	buildManifest := func(suggestedPresentationDelay time.Duration) *core.Manifest {
		periodStart := 5 * time.Hour
		return &core.Manifest{
			MPD: &mpd.MPD{
				Type:                       ptrs.Strptr("dynamic"),
				AvailabilityStartTime:      ptrs.Strptr("2023-01-01T00:00:00Z"),
				PublishTime:                ptrs.Strptr("2023-01-01T06:00:36Z"), // 4 seconds later from latest segment
				SuggestedPresentationDelay: (*mpd.Duration)(&suggestedPresentationDelay),
				Periods: []*mpd.Period{{
					Start: (*mpd.Duration)(&periodStart),
					AdaptationSets: []*mpd.AdaptationSet{{
						SegmentTemplate: &mpd.SegmentTemplate{
							Timescale:              ptrs.Int64ptr(90000),
							PresentationTimeOffset: ptrs.Uint64ptr(10000 * 90000),
							Media:                  ptrs.Strptr("$Time$.mp4"),
							SegmentTimeline: &mpd.SegmentTimeline{
								Segments: []*mpd.SegmentTimelineSegment{
									{
										StartTime:   ptrs.Uint64ptr(13600 * 90000),
										Duration:    4 * 90000,
										RepeatCount: ptrs.Intptr(7),
									},
								},
							},
						},
						Representations: []*mpd.Representation{{}},
					}},
				}},
				UTCTiming: &mpd.DescriptorType{
					SchemeIDURI: ptrs.Strptr("urn:mpeg:dash:utc:direct:2014"),
					Value:       ptrs.Strptr("2023-01-01T06:00:36Z"), // 4 seconds later from latest segment
				},
			},
		}
	}

	t.Run("ok", func(t *testing.T) {
		report := ins.Inspect(buildManifest(7*time.Second), nil)
		require.Equal(t, core.Info, report.Severity)
		assert.Equal(t, "good", report.Message)
	})

	t.Run("ok/without_utc_timing", func(t *testing.T) {
		manifest := buildManifest(7 * time.Second)
		manifest.UTCTiming = nil
		report := ins.Inspect(manifest, nil)
		require.Equal(t, core.Info, report.Severity)
		assert.Equal(t, "good", report.Message)
	})

	t.Run("warn/presentation_time_is_new", func(t *testing.T) {
		report := ins.Inspect(buildManifest(5*time.Second), nil)
		require.Equal(t, core.Warn, report.Severity)
		assert.Equal(t, "latest segment is out of suggested time range", report.Message)
	})

	t.Run("error/presentation_time_is_new", func(t *testing.T) {
		report := ins.Inspect(buildManifest(3*time.Second), nil)
		require.Equal(t, core.Error, report.Severity)
		assert.Equal(t, "latest segment is out of suggested time range", report.Message)
	})

	t.Run("warn/presentation_time_is_old", func(t *testing.T) {
		report := ins.Inspect(buildManifest(35*time.Second), nil)
		require.Equal(t, core.Warn, report.Severity)
		assert.Equal(t, "earliest segment is out of suggested time range", report.Message)
	})

	t.Run("error/presentation_time_is_old", func(t *testing.T) {
		report := ins.Inspect(buildManifest(37*time.Second), nil)
		require.Equal(t, core.Error, report.Severity)
		assert.Equal(t, "earliest segment is out of suggested time range", report.Message)
	})

	t.Run("skip_vod_manifest", func(t *testing.T) {
		report := ins.Inspect(&core.Manifest{MPD: &mpd.MPD{Type: ptrs.Strptr("static")}}, nil)
		require.Equal(t, core.Info, report.Severity)
		assert.Equal(t, "skip VOD manifest", report.Message)
	})

	t.Run("invalid_utc_timing", func(t *testing.T) {
		report := ins.Inspect(&core.Manifest{
			MPD: &mpd.MPD{
				Type: ptrs.Strptr("dynamic"),
				UTCTiming: &mpd.DescriptorType{
					SchemeIDURI: ptrs.Strptr("urn:mpeg:dash:utc:direct:2014"),
					Value:       ptrs.Strptr("9999-99-99T99:99:99Z"),
				},
			},
		}, nil)
		require.Equal(t, core.Error, report.Severity)
		assert.Equal(t, "invalid UTCTiming@value", report.Message)
	})

	t.Run("invalid_publish_time", func(t *testing.T) {
		report := ins.Inspect(&core.Manifest{
			MPD: &mpd.MPD{
				Type:        ptrs.Strptr("dynamic"),
				PublishTime: ptrs.Strptr("9999-99-99T99:99:99Z"),
			},
		}, nil)
		require.Equal(t, core.Error, report.Severity)
		assert.Equal(t, "invalid MPD@publishTime", report.Message)
	})
}
