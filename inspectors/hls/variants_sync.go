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
		Sequence uint64
	}
	type SequenceValue struct {
		MaxDuration float64
		MinDuration float64
	}
	sequenceMap := make(map[SequenceKey]SequenceValue)
	type GroupValue struct {
		MaxSequence uint64
		MinSequence uint64
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
		if media.Alternative != nil {
			groupID = media.Alternative.GroupId
		}
		for _, segment := range media.Segments {
			skey := SequenceKey{
				Sequence: segment.SeqId,
				GroupID:  groupID,
			}
			sval := sequenceMap[skey]
			if sval.MaxDuration == 0 || segment.Duration > sval.MaxDuration {
				sval.MaxDuration = segment.Duration
			}
			if sval.MinDuration == 0 || segment.Duration < sval.MinDuration {
				sval.MinDuration = segment.Duration
			}
			sequenceMap[skey] = sval
		}
		latest := media.Segments[len(media.Segments)-1]
		gval := groupMap[groupID]
		if gval.MaxSequence == 0 || latest.SeqId > gval.MaxSequence {
			gval.MaxSequence = latest.SeqId
		}
		if gval.MinSequence == 0 || latest.SeqId < gval.MinSequence {
			gval.MinSequence = latest.SeqId
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
	var maxSeqDiff uint64
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
	if ins.config.ErrorSequeceDiff != 0 && maxSeqDiff >= uint64(ins.config.ErrorSequeceDiff) {
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
	if ins.config.WarnSequeceDiff != 0 && maxSeqDiff >= uint64(ins.config.WarnSequeceDiff) {
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
