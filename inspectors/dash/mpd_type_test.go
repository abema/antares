package dash

import (
	"testing"

	"github.com/abema/antares/core"
	"github.com/stretchr/testify/require"
	"github.com/zencoder/go-dash/helpers/ptrs"
	"github.com/zencoder/go-dash/mpd"
)

func TestMPDTypeInspector(t *testing.T) {
	ins := NewMPDTypeInspector("dynamic")
	report := ins.Inspect(&core.Manifest{MPD: &mpd.MPD{Type: ptrs.Strptr("dynamic")}}, nil)
	require.Equal(t, core.Info, report.Severity)
	report = ins.Inspect(&core.Manifest{MPD: &mpd.MPD{Type: ptrs.Strptr("static")}}, nil)
	require.Equal(t, core.Error, report.Severity)
}
