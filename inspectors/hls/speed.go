package hls

import (
	"math"
	"time"

	"github.com/abema/antares/core"
	"github.com/abema/antares/inspectors/internal"
	"github.com/grafov/m3u8"
)

type SpeedInspectorConfig struct {
	Interval time.Duration
	Warn     time.Duration
	Error    time.Duration
}

func DefaultSpeedInspectorConfig() *SpeedInspectorConfig {
	return &SpeedInspectorConfig{
		Interval: 10 * time.Minute,
		Warn:     15 * time.Second,
		Error:    30 * time.Second,
	}
}

// NewSpeedInspector returns SpeedInspector.
// It inspects gap between video time and real time.
func NewSpeedInspector() core.HLSInspector {
	return NewSpeedInspectorWithConfig(DefaultSpeedInspectorConfig())
}

func NewSpeedInspectorWithConfig(config *SpeedInspectorConfig) core.HLSInspector {
	return &speedInspector{
		config: config,
		meters: make(map[string]*internal.Speedometer, 8),
	}
}

type speedInspector struct {
	config *SpeedInspectorConfig
	meters map[string]*internal.Speedometer
}

func (ins *speedInspector) Inspect(playlists *core.Playlists, segments core.SegmentStore) *core.Report {
	if playlists.IsVOD() {
		return &core.Report{
			Name:     "SpeedInspector",
			Severity: core.Info,
			Message:  "skip VOD playlist",
		}
	}
	var maxGap float64
	var maxGapURL string
	for _, media := range playlists.MediaPlaylists {
		if media.Closed {
			continue
		}
		realTime := float64(media.Time.UnixNano()) / 1e9
		if len(media.Segments) == 0 {
			return &core.Report{
				Name:     "SpeedInspector",
				Severity: core.Error,
				Message:  "no segments",
				Values:   core.Values{"url": media.URL},
			}
		}
		latest := media.Segments[len(media.Segments)-1]
		meter := ins.meters[media.URL]
		if meter == nil {
			meter = internal.NewSpeedometer(ins.config.Interval.Seconds())
			ins.meters[media.URL] = meter
		}
		// get last time point
		lastTimePoint := meter.LatestTimePoint()
		if lastTimePoint == nil {
			meter.AddTimePoint(&internal.TimePoint{
				RealTime:  realTime,
				SegmentID: latest.SeqId,
			})
			continue
		}
		// calculate accumulated video duration after previous latest segment
		lastSeq := lastTimePoint.SegmentID.(uint64)
		dur := ins.duration(media.Segments, lastSeq+1)
		// add current time point
		meter.AddTimePoint(&internal.TimePoint{
			RealTime:  realTime,
			VideoTime: lastTimePoint.VideoTime + dur,
			SegmentID: latest.SeqId,
		})
		if !meter.Satisfied() {
			continue
		}
		// gap between real time and video time
		gap := meter.Gap()
		if math.Abs(gap) > math.Abs(maxGap) {
			maxGap = gap
			maxGapURL = media.URL
		}
	}
	values := core.Values{"gap": maxGap}
	if maxGapURL != "" {
		values["url"] = maxGapURL
	}
	if ins.config.Error != 0 && math.Abs(maxGap) >= ins.config.Error.Seconds() {
		return &core.Report{
			Name:     "SpeedInspector",
			Severity: core.Error,
			Message:  "large gap between real time and video time",
			Values:   values,
		}
	} else if ins.config.Warn != 0 && math.Abs(maxGap) >= ins.config.Warn.Seconds() {
		return &core.Report{
			Name:     "SpeedInspector",
			Severity: core.Warn,
			Message:  "large gap between real time and video time",
			Values:   values,
		}
	}
	return &core.Report{
		Name:     "SpeedInspector",
		Severity: core.Info,
		Message:  "good",
		Values:   values,
	}
}

func (ins *speedInspector) duration(segments []*m3u8.MediaSegment, begin uint64) float64 {
	var dur float64
	for _, seg := range segments {
		if seg.SeqId >= begin {
			dur += seg.Duration
		}
	}
	return dur
}
