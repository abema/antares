package dash

import (
	"time"

	"github.com/abema/antares/core"
)

type PresentationDelayInspectorConfig struct {
	Warn  time.Duration
	Error time.Duration
}

func DefaultPresentationDelayInspectorConfig() *PresentationDelayInspectorConfig {
	return &PresentationDelayInspectorConfig{
		Warn:  2 * time.Second,
		Error: 0,
	}
}

func NewPresentationDelayInspector() core.DASHInspector {
	return NewPresentationDelayInspectorWithConfig(DefaultPresentationDelayInspectorConfig())
}

func NewPresentationDelayInspectorWithConfig(config *PresentationDelayInspectorConfig) core.DASHInspector {
	return &PresentationDelayInspector{
		config,
	}
}

type PresentationDelayInspector struct {
	config *PresentationDelayInspectorConfig
}

func (ins *PresentationDelayInspector) Inspect(manifest *core.Manifest, segments core.SegmentStore) *core.Report {
	if manifest.Type != nil && *manifest.Type == "static" {
		return &core.Report{
			Name:     "PresentationDelayInspector",
			Severity: core.Info,
			Message:  "skip VOD manifest",
		}
	}

	// get wall-clock
	wallClock := time.Now()
	if manifest.UTCTiming != nil && manifest.UTCTiming.SchemeIDURI != nil && *manifest.UTCTiming.SchemeIDURI == "urn:mpeg:dash:utc:direct:2014" {
		tm, err := time.Parse(time.RFC3339Nano, *manifest.UTCTiming.Value)
		if err != nil {
			return &core.Report{
				Name:     "PresentationDelayInspector",
				Severity: core.Error,
				Message:  "invalid UTCTiming@value",
				Values:   core.Values{"error": err},
			}
		}
		wallClock = tm
	} else if manifest.PublishTime != nil {
		tm, err := time.Parse(time.RFC3339Nano, *manifest.PublishTime)
		if err != nil {
			return &core.Report{
				Name:     "PresentationDelayInspector",
				Severity: core.Error,
				Message:  "invalid MPD@publishTime",
				Values:   core.Values{"error": err},
			}
		}
		wallClock = tm
	}

	// get suggestedPresentationDelay
	var suggestedPresentationDelay time.Duration
	if manifest.SuggestedPresentationDelay != nil {
		suggestedPresentationDelay = time.Duration(*manifest.SuggestedPresentationDelay)
	}

	// get availabilityStartTime
	var availabilityStartTime time.Time
	if manifest.AvailabilityStartTime != nil {
		tm, err := time.Parse(time.RFC3339Nano, *manifest.AvailabilityStartTime)
		if err != nil {
			return &core.Report{
				Name:     "PresentationDelayInspector",
				Severity: core.Error,
				Message:  "invalid MPD@availabilityStartTime",
				Values:   core.Values{"error": err},
			}
		}
		availabilityStartTime = tm
	}

	// get latest time
	var earliestVideoTime time.Time
	var latestVideoTime time.Time
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
		s := int64(segment.Time) - int64(offset)
		fs := periodStart + float64(s)/timescale
		ts := availabilityStartTime.Add(time.Duration(fs*1e9) * time.Nanosecond)
		if earliestVideoTime.IsZero() || ts.Before(earliestVideoTime) {
			earliestVideoTime = ts
		}
		e := s + int64(segment.Duration)
		fe := periodStart + float64(e)/timescale
		te := availabilityStartTime.Add(time.Duration(fe*1e9) * time.Nanosecond)
		if te.After(latestVideoTime) {
			latestVideoTime = te
		}
		return true
	})
	if err != nil {
		return &core.Report{
			Name:     "PresentationDelayInspector",
			Severity: core.Error,
			Message:  "unexpected error",
			Values:   core.Values{"error": err},
		}
	}

	values := core.Values{
		"earliestVideoTime":          earliestVideoTime.UTC().Format(time.RFC3339Nano),
		"latestVideoTime":            latestVideoTime.UTC().Format(time.RFC3339Nano),
		"wallClock":                  wallClock.UTC().Format(time.RFC3339Nano),
		"suggestedPresentationDelay": suggestedPresentationDelay,
	}
	earliestRenderTime := earliestVideoTime.Add(suggestedPresentationDelay)
	latestRenderTime := latestVideoTime.Add(suggestedPresentationDelay)
	if earliestRenderTime.Add(ins.config.Error).After(wallClock) {
		return &core.Report{
			Name:     "PresentationDelayInspector",
			Severity: core.Error,
			Message:  "earliest segment is out of suggested time range",
			Values:   values,
		}
	} else if earliestRenderTime.Add(ins.config.Warn).After(wallClock) {
		return &core.Report{
			Name:     "PresentationDelayInspector",
			Severity: core.Warn,
			Message:  "earliest segment is out of suggested time range",
			Values:   values,
		}
	}
	if latestRenderTime.Add(-ins.config.Error).Before(wallClock) {
		return &core.Report{
			Name:     "PresentationDelayInspector",
			Severity: core.Error,
			Message:  "latest segment is out of suggested time range",
			Values:   values,
		}
	} else if latestRenderTime.Add(-ins.config.Warn).Before(wallClock) {
		return &core.Report{
			Name:     "PresentationDelayInspector",
			Severity: core.Warn,
			Message:  "latest segment is out of suggested time range",
			Values:   values,
		}
	}
	return &core.Report{
		Name:     "PresentationDelayInspector",
		Severity: core.Info,
		Message:  "good",
		Values:   values,
	}
}
