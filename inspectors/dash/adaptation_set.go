package dash

import (
	"fmt"

	"github.com/abema/antares/core"
	"github.com/abema/antares/internal/strings"
)

type AdaptationSetInspectorConfig struct {
	MandatoryMimeTypes []string
	ValidMimeTypes     []string
}

// NewAdaptationSetInspector returns AdaptationSetInspector.
// It inspects number of AdaptationSets and those attributes.
func NewAdaptationSetInspector(config *AdaptationSetInspectorConfig) core.DASHInspector {
	return &adaptationSetInspector{
		config: config,
	}
}

type adaptationSetInspector struct {
	config *AdaptationSetInspectorConfig
}

func (ins *adaptationSetInspector) Inspect(manifest *core.Manifest, segments core.SegmentStore) *core.Report {
	var noMimeType bool
	mimeTypeSet := make(map[string]struct{}, 4)
	for _, period := range manifest.Periods {
		for _, adaptationSet := range period.AdaptationSets {
			if adaptationSet.MimeType == nil {
				noMimeType = true
			} else {
				mimeTypeSet[*adaptationSet.MimeType] = struct{}{}
			}
		}
	}
	mimeTypes := make([]string, 0, len(mimeTypeSet))
	for mimeType := range mimeTypeSet {
		mimeTypes = append(mimeTypes, mimeType)
	}
	values := core.Values{
		"mimeType": mimeTypes,
	}

	if noMimeType {
		return &core.Report{
			Name:     "AdaptationSetInspector",
			Severity: core.Error,
			Message:  "mimeType attribute is omitted",
			Values:   values,
		}
	}
	for mimeType := range mimeTypeSet {
		if !strings.ContainsIn(mimeType, ins.config.MandatoryMimeTypes) &&
			!strings.ContainsIn(mimeType, ins.config.ValidMimeTypes) {
			return &core.Report{
				Name:     "AdaptationSetInspector",
				Severity: core.Error,
				Message:  fmt.Sprintf("invalid mimeType [%s]", mimeType),
				Values:   values,
			}
		}
	}

	for _, period := range manifest.Periods {
		mimeTypeSetInPeriod := make(map[string]struct{}, 4)
		for _, adaptationSet := range period.AdaptationSets {
			mimeTypeSetInPeriod[*adaptationSet.MimeType] = struct{}{}
		}
		for _, mimeType := range ins.config.MandatoryMimeTypes {
			if _, ok := mimeTypeSetInPeriod[mimeType]; !ok {
				return &core.Report{
					Name:     "AdaptationSetInspector",
					Severity: core.Error,
					Message:  fmt.Sprintf("mimeType [%s] is mandatory", mimeType),
					Values:   values,
				}
			}
		}
	}
	return &core.Report{
		Name:     "AdaptationSetInspector",
		Severity: core.Info,
		Message:  "good",
		Values:   values,
	}
}
