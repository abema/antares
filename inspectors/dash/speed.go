package dash

import (
	"math"
	"time"

	"github.com/abema/antares/core"
	"github.com/abema/antares/inspectors/internal"
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
func NewSpeedInspector() core.DASHInspector {
	return NewSpeedInspectorWithConfig(DefaultSpeedInspectorConfig())
}

func NewSpeedInspectorWithConfig(config *SpeedInspectorConfig) core.DASHInspector {
	return &speedInspector{
		config: config,
		meter:  internal.NewSpeedometer(config.Interval.Seconds()),
	}
}

type speedInspector struct {
	config *SpeedInspectorConfig
	meter  *internal.Speedometer
}

func (ins *speedInspector) Inspect(manifest *core.Manifest, segments core.SegmentStore) *core.Report {
	if manifest.Type != nil && *manifest.Type == "static" {
		return &core.Report{
			Name:     "SpeedInspector",
			Severity: core.Info,
			Message:  "skip VOD manifest",
		}
	}
	var videoTime float64
	err := manifest.EachSegments(func(segment *core.DASHSegment) (cont bool) {
		if segment.Initialization {
			return true
		}
		var periodStart float64
		if segment.Period.Start != nil {
			periodStart = time.Duration(*segment.Period.Start).Seconds()
		}
		var offset uint64
		if segment.SegmentTemplate.PresentationTimeOffset != nil {
			offset = *segment.SegmentTemplate.PresentationTimeOffset
		}
		timescale := float64(1)
		if segment.SegmentTemplate.Timescale != nil {
			timescale = float64(*segment.SegmentTemplate.Timescale)
		}
		t := int64(segment.Time) - int64(offset) + int64(segment.Duration)
		vt := periodStart + float64(t)/timescale
		if vt > videoTime {
			videoTime = vt
		}
		return true
	})
	if err != nil {
		return &core.Report{
			Name:     "SpeedInspector",
			Severity: core.Error,
			Message:  "unexpected error",
			Values:   core.Values{"error": err},
		}
	}
	ins.meter.AddTimePoint(&internal.TimePoint{
		RealTime:  float64(manifest.Time.UnixNano()) / 1e9,
		VideoTime: videoTime,
	})
	if !ins.meter.Satisfied() {
		return &core.Report{
			Name:     "SpeedInspector",
			Severity: core.Info,
			Message:  "wait for accumulating history",
		}
	}
	gap := ins.meter.Gap()
	values := core.Values{
		"gap":       gap,
		"realTime":  ins.meter.RealTimeElapsed(),
		"videoTime": ins.meter.VideoTimeElapsed(),
	}
	if ins.config.Error != 0 && math.Abs(gap) >= ins.config.Error.Seconds() {
		return &core.Report{
			Name:     "SpeedInspector",
			Severity: core.Error,
			Message:  "large gap between real time and video time",
			Values:   values,
		}
	} else if ins.config.Warn != 0 && math.Abs(gap) >= ins.config.Warn.Seconds() {
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
