package dash

import (
	"fmt"

	"github.com/abema/antares/core"
)

// NewMPDTypeInspector returns MPDTypeInspector.
// It inspects whether MPD@type equals to mpdType.
func NewMPDTypeInspector(mpdType string) core.DASHInspector {
	return &mpdTypeInspector{
		mpdType: mpdType,
	}
}

type mpdTypeInspector struct {
	mpdType string
}

func (ins *mpdTypeInspector) Inspect(manifest *core.Manifest, segments core.SegmentStore) *core.Report {
	mpdType := "static"
	if manifest.Type != nil {
		mpdType = *manifest.Type
	}

	values := core.Values{
		"type": mpdType,
	}

	if mpdType != ins.mpdType {
		return &core.Report{
			Name:     "MPDTypeInspector",
			Severity: core.Error,
			Message:  fmt.Sprintf("invalid Type [%s]", mpdType),
			Values:   values,
		}
	}
	return &core.Report{
		Name:     "MPDTypeInspector",
		Severity: core.Info,
		Message:  "good",
		Values:   values,
	}
}
