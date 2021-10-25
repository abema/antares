package dash

import (
	"testing"

	"github.com/abema/antares/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zencoder/go-dash/helpers/ptrs"
	"github.com/zencoder/go-dash/mpd"
)

func TestNewRepresentationInspector(t *testing.T) {
	ins := NewRepresentationInspector(&RepresentationInspectorConfig{
		ErrorMaxHeight:         720,
		WarnMaxHeight:          480,
		WarnMinHeight:          240,
		ErrorMinHeight:         180,
		ValidPARs:              []AspectRatio{{X: 16, Y: 9}},
		ErrorMaxVideoBandwidth: 2000 * 1e3,
		WarnMaxVideoBandwidth:  1000 * 1e3,
		WarnMinVideoBandwidth:  200 * 1e3,
		ErrorMinVideoBandwidth: 100 * 1e3,
		ErrorMaxAudioBandwidth: 200 * 1e3,
		WarnMaxAudioBandwidth:  100 * 1e3,
		WarnMinAudioBandwidth:  50 * 1e3,
		ErrorMinAudioBandwidth: 10 * 1e3,
	})

	testCases := []struct {
		name            string
		highVideoWidth  int64
		highVideoHeight int64
		lowVideoWidth   int64
		lowVideoHeight  int64
		highVideo       int64
		lowVideo        int64
		highAudio       int64
		lowAudio        int64
		severity        core.Severity
		message         string
	}{
		{
			name:           "ok",
			highVideoWidth: 853, highVideoHeight: 480,
			lowVideoWidth: 426, lowVideoHeight: 240,
			highVideo: 1000 * 1e3, lowVideo: 200 * 1e3,
			highAudio: 100 * 1e3, lowAudio: 50 * 1e3,
			severity: core.Info,
			message:  "good",
		},
		{
			name:           "large-hight/warn",
			highVideoWidth: 1280, highVideoHeight: 720,
			lowVideoWidth: 426, lowVideoHeight: 240,
			highVideo: 1000 * 1e3, lowVideo: 200 * 1e3,
			highAudio: 100 * 1e3, lowAudio: 50 * 1e3,
			severity: core.Warn,
			message:  "too large height",
		},
		{
			name:           "large-hight/error",
			highVideoWidth: 1920, highVideoHeight: 1080,
			lowVideoWidth: 426, lowVideoHeight: 240,
			highVideo: 1000 * 1e3, lowVideo: 200 * 1e3,
			highAudio: 100 * 1e3, lowAudio: 50 * 1e3,
			severity: core.Error,
			message:  "too large height",
		},
		{
			name:           "small-hight/warn",
			highVideoWidth: 853, highVideoHeight: 480,
			lowVideoWidth: 320, lowVideoHeight: 180,
			highVideo: 1000 * 1e3, lowVideo: 200 * 1e3,
			highAudio: 100 * 1e3, lowAudio: 50 * 1e3,
			severity: core.Warn,
			message:  "too small height",
		},
		{
			name:           "small-hight/error",
			highVideoWidth: 853, highVideoHeight: 480,
			lowVideoWidth: 178, lowVideoHeight: 100,
			highVideo: 1000 * 1e3, lowVideo: 200 * 1e3,
			highAudio: 100 * 1e3, lowAudio: 50 * 1e3,
			severity: core.Error,
			message:  "too small height",
		},
		{
			name:           "video-high-bitrate/warn",
			highVideoWidth: 853, highVideoHeight: 480,
			lowVideoWidth: 426, lowVideoHeight: 240,
			highVideo: 2000 * 1e3, lowVideo: 200 * 1e3,
			highAudio: 100 * 1e3, lowAudio: 50 * 1e3,
			severity: core.Warn,
			message:  "high video bandwidth",
		},
		{
			name:           "video-high-bitrate/error",
			highVideoWidth: 853, highVideoHeight: 480,
			lowVideoWidth: 426, lowVideoHeight: 240,
			highVideo: 4000 * 1e3, lowVideo: 200 * 1e3,
			highAudio: 100 * 1e3, lowAudio: 50 * 1e3,
			severity: core.Error,
			message:  "high video bandwidth",
		},
		{
			name:           "video-low-bitrate/warn",
			highVideoWidth: 853, highVideoHeight: 480,
			lowVideoWidth: 426, lowVideoHeight: 240,
			highVideo: 1000 * 1e3, lowVideo: 100 * 1e3,
			highAudio: 100 * 1e3, lowAudio: 50 * 1e3,
			severity: core.Warn,
			message:  "low video bandwidth",
		},
		{
			name:           "video-low-bitrate/error",
			highVideoWidth: 853, highVideoHeight: 480,
			lowVideoWidth: 426, lowVideoHeight: 240,
			highVideo: 1000 * 1e3, lowVideo: 50 * 1e3,
			highAudio: 100 * 1e3, lowAudio: 50 * 1e3,
			severity: core.Error,
			message:  "low video bandwidth",
		},
		{
			name:           "audio-high-bitrate/warn",
			highVideoWidth: 853, highVideoHeight: 480,
			lowVideoWidth: 426, lowVideoHeight: 240,
			highVideo: 1000 * 1e3, lowVideo: 200 * 1e3,
			highAudio: 200 * 1e3, lowAudio: 50 * 1e3,
			severity: core.Warn,
			message:  "high audio bandwidth",
		},
		{
			name:           "audio-high-bitrate/error",
			highVideoWidth: 853, highVideoHeight: 480,
			lowVideoWidth: 426, lowVideoHeight: 240,
			highVideo: 1000 * 1e3, lowVideo: 200 * 1e3,
			highAudio: 300 * 1e3, lowAudio: 50 * 1e3,
			severity: core.Error,
			message:  "high audio bandwidth",
		},
		{
			name:           "audio-low-bitrate/warn",
			highVideoWidth: 853, highVideoHeight: 480,
			lowVideoWidth: 426, lowVideoHeight: 240,
			highVideo: 1000 * 1e3, lowVideo: 200 * 1e3,
			highAudio: 100 * 1e3, lowAudio: 10 * 1e3,
			severity: core.Warn,
			message:  "low audio bandwidth",
		},
		{
			name:           "audio-low-bitrate/error",
			highVideoWidth: 853, highVideoHeight: 480,
			lowVideoWidth: 426, lowVideoHeight: 240,
			highVideo: 1000 * 1e3, lowVideo: 200 * 1e3,
			highAudio: 100 * 1e3, lowAudio: 5 * 1e3,
			severity: core.Error,
			message:  "low audio bandwidth",
		},
		{
			name:           "aspect-ratio/error",
			highVideoWidth: 640, highVideoHeight: 480,
			lowVideoWidth: 426, lowVideoHeight: 240,
			highVideo: 1000 * 1e3, lowVideo: 200 * 1e3,
			highAudio: 100 * 1e3, lowAudio: 50 * 1e3,
			severity: core.Error,
			message:  "invalid PAR: width=640 height=480 sar=[1:1]",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			report := ins.Inspect(&core.Manifest{
				MPD: &mpd.MPD{
					Periods: []*mpd.Period{{
						AdaptationSets: []*mpd.AdaptationSet{{
							CommonAttributesAndElements: mpd.CommonAttributesAndElements{
								MimeType: ptrs.Strptr("video/mp4"),
							},
							Representations: []*mpd.Representation{
								{Width: ptrs.Int64ptr(tc.highVideoWidth), Height: ptrs.Int64ptr(tc.highVideoHeight), Bandwidth: ptrs.Int64ptr(tc.highVideo)},
								{Width: ptrs.Int64ptr(tc.lowVideoWidth), Height: ptrs.Int64ptr(tc.lowVideoHeight), Bandwidth: ptrs.Int64ptr(tc.lowVideo)},
							},
						}, {
							CommonAttributesAndElements: mpd.CommonAttributesAndElements{
								MimeType: ptrs.Strptr("audio/mp4"),
							},
							Representations: []*mpd.Representation{
								{Bandwidth: ptrs.Int64ptr(tc.highAudio)},
								{Bandwidth: ptrs.Int64ptr(tc.lowAudio)},
							},
						}},
					}},
				},
			}, nil)
			require.Equal(t, tc.severity, report.Severity)
			require.Equal(t, tc.message, report.Message)
		})
	}
}

func TestNewRepresentationInspector_Omition(t *testing.T) {
	testCases := []struct {
		name                string
		allowHeightOmittion bool
		allowWidthOmittion  bool
		mimeType            *string
		bandwidth           *int64
		width               *int64
		height              *int64
		message             string
	}{
		{
			name:                "height-omittion-error",
			allowHeightOmittion: false,
			allowWidthOmittion:  true,
			mimeType:            ptrs.Strptr("video/mp4"),
			bandwidth:           ptrs.Int64ptr(100 * 1e3),
			message:             "height attribute is omitted",
		},
		{
			name:                "width-omittion-error",
			allowHeightOmittion: true,
			allowWidthOmittion:  false,
			mimeType:            ptrs.Strptr("video/mp4"),
			bandwidth:           ptrs.Int64ptr(100 * 1e3),
			message:             "width attribute is omitted",
		},
		{
			name:                "bandwidth-omittion-error",
			allowHeightOmittion: true,
			allowWidthOmittion:  true,
			mimeType:            ptrs.Strptr("video/mp4"),
			message:             "bandwidth attribute is omitted",
		},
		{
			name:                "mimeType-omittion-error",
			allowHeightOmittion: true,
			allowWidthOmittion:  true,
			bandwidth:           ptrs.Int64ptr(100 * 1e3),
			message:             "mimeType attribute is omitted",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ins := NewRepresentationInspector(&RepresentationInspectorConfig{
				AllowHeightOmittion: tc.allowHeightOmittion,
				AllowWidthOmittion:  tc.allowWidthOmittion,
			})
			report := ins.Inspect(&core.Manifest{
				MPD: &mpd.MPD{
					Periods: []*mpd.Period{{
						AdaptationSets: []*mpd.AdaptationSet{{
							CommonAttributesAndElements: mpd.CommonAttributesAndElements{
								MimeType: tc.mimeType,
							},
							Representations: []*mpd.Representation{{Bandwidth: tc.bandwidth}},
						}},
					}},
				},
			}, nil)
			require.Equal(t, core.Error, report.Severity)
			require.Equal(t, tc.message, report.Message)
		})
	}
}

func TestGetResolution(t *testing.T) {
	t.Run("attribute of AdaptationSet", func(t *testing.T) {
		rsl, err := getResolution(&mpd.AdaptationSet{
			CommonAttributesAndElements: mpd.CommonAttributesAndElements{
				Width:  ptrs.Strptr("1920"),
				Height: ptrs.Strptr("1080"),
				Sar:    ptrs.Strptr("1:1"),
			},
		}, &mpd.Representation{})
		require.NoError(t, err)
		assert.Equal(t, int64(1920), *rsl.Width)
		assert.Equal(t, int64(1080), *rsl.Height)
		assert.Equal(t, AspectRatio{X: 1, Y: 1}, rsl.SAR)
	})

	t.Run("attribute of Representation", func(t *testing.T) {
		rsl, err := getResolution(&mpd.AdaptationSet{
			CommonAttributesAndElements: mpd.CommonAttributesAndElements{
				Width:  ptrs.Strptr("1920"),
				Height: ptrs.Strptr("1080"),
				Sar:    ptrs.Strptr("1:1"),
			},
		}, &mpd.Representation{
			CommonAttributesAndElements: mpd.CommonAttributesAndElements{
				Sar: ptrs.Strptr("1:1"),
			},
			Width:  ptrs.Int64ptr(1280),
			Height: ptrs.Int64ptr(720),
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1280), *rsl.Width)
		assert.Equal(t, int64(720), *rsl.Height)
		assert.Equal(t, AspectRatio{X: 1, Y: 1}, rsl.SAR)
	})
}
