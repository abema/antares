package hls

import (
	"time"

	"github.com/abema/antares/core"
)

type VariantsSyncInspectorConfig struct {
	WarnSegmentDurationDiff  time.Duration
	ErrorSegmentDurationDiff time.Duration
	WarnSequeceDiff          uint
	ErrorSequeceDiff         uint
}

func DefaultVariantsSyncInspectorConfig() *VariantsSyncInspectorConfig {
	return &VariantsSyncInspectorConfig{
		WarnSegmentDurationDiff:  500 * time.Millisecond,
		ErrorSegmentDurationDiff: 1000 * time.Millisecond,
		WarnSequeceDiff:          2,
		ErrorSequeceDiff:         4,
	}
}

// NewVariantsSyncInspector returns VariantsSyncInspector.
// It inspects synchronization of variant streams.
func NewVariantsSyncInspector() core.HLSInspector {
	return NewVariantsSyncInspectorWithConfig(DefaultVariantsSyncInspectorConfig())
}

func NewVariantsSyncInspectorWithConfig(config *VariantsSyncInspectorConfig) core.HLSInspector {
	return &variantsSyncInspector{
		config: config,
	}
}

type variantsSyncInspector struct {
	config *VariantsSyncInspectorConfig
}

func (ins *variantsSyncInspector) Inspect(playlists *core.Playlists, _ core.SegmentStore) *core.Report {
	type SequenceKey struct {
		GroupID  string
		Sequence int64
	}
	type SequenceValue struct {
		MaxDuration float64
		MinDuration float64
	}
	sequenceMap := make(map[SequenceKey]SequenceValue)
	type GroupValue struct {
		MaxSequence int64
		MinSequence int64
	}
	groupMap := make(map[string]GroupValue)
	for _, media := range playlists.MediaPlaylists {
		if len(media.Segments) == 0 {
			return &core.Report{
				Name:     "VariantsSyncInspector",
				Severity: core.Info,
				Message:  "no segments",
			}
		}
		var groupID string
		if media.MediaAttrs != nil {
			groupID = media.MediaAttrs.GroupID()
		}
		for _, segment := range media.Segments {
			skey := SequenceKey{
				Sequence: segment.Sequence,
				GroupID:  groupID,
			}
			sval := sequenceMap[skey]
			if sval.MaxDuration == 0 || segment.Tags.ExtInfValue() > sval.MaxDuration {
				sval.MaxDuration = segment.Tags.ExtInfValue()
			}
			if sval.MinDuration == 0 || segment.Tags.ExtInfValue() < sval.MinDuration {
				sval.MinDuration = segment.Tags.ExtInfValue()
			}
			sequenceMap[skey] = sval
		}
		latest := media.Segments[len(media.Segments)-1]
		gval := groupMap[groupID]
		if gval.MaxSequence == 0 || latest.Sequence > gval.MaxSequence {
			gval.MaxSequence = latest.Sequence
		}
		if gval.MinSequence == 0 || latest.Sequence < gval.MinSequence {
			gval.MinSequence = latest.Sequence
		}
		groupMap[groupID] = gval
	}
	var maxDurDiff float64
	for _, sval := range sequenceMap {
		durDiff := sval.MaxDuration - sval.MinDuration
		if durDiff > maxDurDiff {
			maxDurDiff = durDiff
		}
	}
	var maxSeqDiff int64
	for _, gval := range groupMap {
		seqDiff := gval.MaxSequence - gval.MinSequence
		if seqDiff > maxSeqDiff {
			maxSeqDiff = seqDiff
		}
	}
	values := core.Values{"durDiff": maxDurDiff, "seqDiff": maxSeqDiff}
	if ins.config.ErrorSegmentDurationDiff != 0 && maxDurDiff >= ins.config.ErrorSegmentDurationDiff.Seconds() {
		return &core.Report{
			Name:     "VariantsSyncInspector",
			Severity: core.Error,
			Message:  "large duration difference",
			Values:   values,
		}
	}
	if ins.config.ErrorSequeceDiff != 0 && maxSeqDiff >= int64(ins.config.ErrorSequeceDiff) {
		return &core.Report{
			Name:     "VariantsSyncInspector",
			Severity: core.Error,
			Message:  "large sequence difference",
			Values:   values,
		}
	}
	if ins.config.WarnSegmentDurationDiff != 0 && maxDurDiff >= ins.config.WarnSegmentDurationDiff.Seconds() {
		return &core.Report{
			Name:     "VariantsSyncInspector",
			Severity: core.Warn,
			Message:  "large duration difference",
			Values:   values,
		}
	}
	if ins.config.WarnSequeceDiff != 0 && maxSeqDiff >= int64(ins.config.WarnSequeceDiff) {
		return &core.Report{
			Name:     "VariantsSyncInspector",
			Severity: core.Warn,
			Message:  "large sequence difference",
			Values:   values,
		}
	}
	return &core.Report{
		Name:     "VariantsSyncInspector",
		Severity: core.Info,
		Message:  "good",
		Values:   values,
	}
}
