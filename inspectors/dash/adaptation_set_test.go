package dash

import (
	"testing"

	"github.com/abema/antares/core"
	"github.com/stretchr/testify/require"
	"github.com/zencoder/go-dash/helpers/ptrs"
	"github.com/zencoder/go-dash/mpd"
)

func Test(t *testing.T) {
	ins := NewAdaptationSetInspector(&AdaptationSetInspectorConfig{
		MandatoryMimeTypes: []string{"video/mp4"},
		ValidMimeTypes:     []string{"audio/mp4"},
	})

	t.Run("video_audio/ok", func(t *testing.T) {
		report := ins.Inspect(&core.Manifest{
			MPD: &mpd.MPD{
				Periods: []*mpd.Period{{
					AdaptationSets: []*mpd.AdaptationSet{{
						CommonAttributesAndElements: mpd.CommonAttributesAndElements{
							MimeType: ptrs.Strptr("video/mp4"),
						},
					}, {
						CommonAttributesAndElements: mpd.CommonAttributesAndElements{
							MimeType: ptrs.Strptr("audio/mp4"),
						},
					}},
				}},
			},
		}, nil)
		require.Equal(t, core.Info, report.Severity)
	})

	t.Run("video/ok", func(t *testing.T) {
		report := ins.Inspect(&core.Manifest{
			MPD: &mpd.MPD{
				Periods: []*mpd.Period{{
					AdaptationSets: []*mpd.AdaptationSet{{
						CommonAttributesAndElements: mpd.CommonAttributesAndElements{
							MimeType: ptrs.Strptr("video/mp4"),
						},
					}},
				}},
			},
		}, nil)
		require.Equal(t, core.Info, report.Severity)
	})

	t.Run("video_text/error", func(t *testing.T) {
		report := ins.Inspect(&core.Manifest{
			MPD: &mpd.MPD{
				Periods: []*mpd.Period{{
					AdaptationSets: []*mpd.AdaptationSet{{
						CommonAttributesAndElements: mpd.CommonAttributesAndElements{
							MimeType: ptrs.Strptr("video/mp4"),
						},
					}, {
						CommonAttributesAndElements: mpd.CommonAttributesAndElements{
							MimeType: ptrs.Strptr("text/vtt"),
						},
					}},
				}},
			},
		}, nil)
		require.Equal(t, core.Error, report.Severity)
	})
}
