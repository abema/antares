package dash

import (
	"testing"
	"time"

	"github.com/abema/antares/core"
	"github.com/stretchr/testify/require"
	"github.com/zencoder/go-dash/helpers/ptrs"
	"github.com/zencoder/go-dash/mpd"
)

func TestSpeedInspectorTest(t *testing.T) {
	manifest := func(typ string, t int64, start uint64) *core.Manifest {
		return &core.Manifest{
			Time: time.Unix(t, 0),
			MPD: &mpd.MPD{
				Type: ptrs.Strptr(typ),
				Periods: []*mpd.Period{
					{
						Start: (*mpd.Duration)(ptrs.Int64ptr(0)),
						AdaptationSets: []*mpd.AdaptationSet{
							{
								SegmentTemplate: &mpd.SegmentTemplate{
									Timescale:              ptrs.Int64ptr(90000),
									PresentationTimeOffset: ptrs.Uint64ptr(1100 * 90000),
									Initialization:         ptrs.Strptr("init.mp4"),
									Media:                  ptrs.Strptr("$Time$.mp4"),
									SegmentTimeline: &mpd.SegmentTimeline{
										Segments: []*mpd.SegmentTimelineSegment{
											{
												StartTime:   ptrs.Uint64ptr(start * 90000),
												Duration:    10 * 90000,
												RepeatCount: ptrs.Intptr(10),
											},
										},
									},
								},
								Representations: []*mpd.Representation{{}},
							},
						},
					},
				},
			},
		}
	}
	ins := NewSpeedInspector()
	rep := ins.Inspect(manifest("dynamic", 1000, 1000), nil)
	require.Equal(t, core.Info, rep.Severity)
	rep = ins.Inspect(manifest("dynamic", 1050, 1050), nil)
	require.Equal(t, core.Info, rep.Severity)
	rep = ins.Inspect(manifest("dynamic", 1100, 1100), nil)
	require.Equal(t, core.Info, rep.Severity)
	rep = ins.Inspect(manifest("dynamic", 1150, 1130), nil)
	require.Equal(t, core.Warn, rep.Severity)
	rep = ins.Inspect(manifest("dynamic", 1200, 1170), nil)
	require.Equal(t, core.Error, rep.Severity)
	rep = ins.Inspect(manifest("static", 1200, 1170), nil)
	require.Equal(t, core.Info, rep.Severity)
}
