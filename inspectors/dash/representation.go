package dash

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/abema/antares/core"
	"github.com/zencoder/go-dash/mpd"
)

type AspectRatio struct {
	X int64
	Y int64
}

func ParseAspectRatio(ar string) (AspectRatio, error) {
	s := strings.SplitN(ar, ":", 2)
	if len(s) != 2 {
		return AspectRatio{}, fmt.Errorf("invalid aspect ratio format: %s", ar)
	}
	x, err := strconv.ParseInt(s[0], 10, 64)
	if err != nil {
		return AspectRatio{}, fmt.Errorf("invalid aspect ratio format: %s", ar)
	}
	y, err := strconv.ParseInt(s[1], 10, 64)
	if err != nil {
		return AspectRatio{}, fmt.Errorf("invalid aspect ratio format: %s", ar)
	}
	return AspectRatio{X: x, Y: y}, nil
}

type RepresentationInspectorConfig struct {
	WarnMaxHeight          int64
	ErrorMaxHeight         int64
	WarnMinHeight          int64
	ErrorMinHeight         int64
	ValidPARs              []AspectRatio
	AllowHeightOmittion    bool
	AllowWidthOmittion     bool
	WarnMaxVideoBandwidth  int64
	ErrorMaxVideoBandwidth int64
	WarnMinVideoBandwidth  int64
	ErrorMinVideoBandwidth int64
	WarnMaxAudioBandwidth  int64
	ErrorMaxAudioBandwidth int64
	WarnMinAudioBandwidth  int64
	ErrorMinAudioBandwidth int64
}

func NewRepresentationInspector(config *RepresentationInspectorConfig) core.DASHInspector {
	return &representationInspector{
		config: config,
	}
}

type representationInspector struct {
	config *RepresentationInspectorConfig
}

func (ins *representationInspector) Inspect(manifest *core.Manifest, segments core.SegmentStore) *core.Report {
	rsls := make([]*resolution, 0)
	var maxVideoBandwidth int64
	minVideoBandwidth := int64(math.MaxInt64)
	var maxAudioBandwidth int64
	minAudioBandwidth := int64(math.MaxInt64)

	for _, period := range manifest.Periods {
		for _, adaptationSet := range period.AdaptationSets {
			if adaptationSet.MimeType == nil {
				return &core.Report{
					Name:     "RepresentationInspector",
					Severity: core.Error,
					Message:  "mimeType attribute is omitted",
				}
			}
			if len(adaptationSet.Representations) == 0 {
				return &core.Report{
					Name:     "RepresentationInspector",
					Severity: core.Error,
					Message:  "no representation tag",
				}
			}
			for _, representation := range adaptationSet.Representations {
				if representation.Bandwidth == nil {
					return &core.Report{
						Name:     "RepresentationInspector",
						Severity: core.Error,
						Message:  "bandwidth attribute is omitted",
					}
				}
				switch *adaptationSet.MimeType {
				case "video/mp4":
					rsl, err := getResolution(adaptationSet, representation)
					if err != nil {
						return &core.Report{
							Name:     "RepresentationInspector",
							Severity: core.Error,
							Message:  err.Error(),
						}
					}
					rsls = append(rsls, rsl)
					if *representation.Bandwidth > maxVideoBandwidth {
						maxVideoBandwidth = *representation.Bandwidth
					}
					if *representation.Bandwidth < minVideoBandwidth {
						minVideoBandwidth = *representation.Bandwidth
					}
				case "audio/mp4":
					if *representation.Bandwidth > maxAudioBandwidth {
						maxAudioBandwidth = *representation.Bandwidth
					}
					if *representation.Bandwidth < minAudioBandwidth {
						minAudioBandwidth = *representation.Bandwidth
					}
				}
			}
		}
	}

	values := core.Values{
		"maxVideoBandwidth": maxVideoBandwidth,
		"minVideoBandwidth": minVideoBandwidth,
		"maxAudioBandwidth": maxAudioBandwidth,
		"minAudioBandwidth": minAudioBandwidth,
	}

	for _, rsl := range rsls {
		if rsl.Height == nil {
			if !ins.config.AllowHeightOmittion {
				return &core.Report{
					Name:     "RepresentationInspector",
					Severity: core.Error,
					Message:  "height attribute is omitted",
					Values:   values,
				}
			}
		} else if ins.config.ErrorMaxHeight != 0 && *rsl.Height > ins.config.ErrorMaxHeight {
			return &core.Report{
				Name:     "RepresentationInspector",
				Severity: core.Error,
				Message:  "too large height",
				Values:   values,
			}
		} else if ins.config.WarnMaxHeight != 0 && *rsl.Height > ins.config.WarnMaxHeight {
			return &core.Report{
				Name:     "RepresentationInspector",
				Severity: core.Warn,
				Message:  "too large height",
				Values:   values,
			}
		} else if ins.config.ErrorMinHeight != 0 && *rsl.Height < ins.config.ErrorMinHeight {
			return &core.Report{
				Name:     "RepresentationInspector",
				Severity: core.Error,
				Message:  "too small height",
				Values:   values,
			}
		} else if ins.config.WarnMinHeight != 0 && *rsl.Height < ins.config.WarnMinHeight {
			return &core.Report{
				Name:     "RepresentationInspector",
				Severity: core.Warn,
				Message:  "too small height",
				Values:   values,
			}
		}
		if rsl.Width == nil && !ins.config.AllowWidthOmittion {
			return &core.Report{
				Name:     "RepresentationInspector",
				Severity: core.Error,
				Message:  "width attribute is omitted",
				Values:   values,
			}
		}
		if rsl.Width != nil && rsl.Height != nil && len(ins.config.ValidPARs) != 0 {
			if !containsAspectRatio(AspectRatio{
				X: rsl.SAR.X * (*rsl.Width),
				Y: rsl.SAR.Y * (*rsl.Height),
			}, ins.config.ValidPARs) {
				return &core.Report{
					Name:     "RepresentationInspector",
					Severity: core.Error,
					Message: fmt.Sprintf("invalid PAR: width=%d height=%d sar=[%d:%d]",
						*rsl.Width, *rsl.Height, rsl.SAR.X, rsl.SAR.Y),
					Values: values,
				}
			}
		}
	}
	if ins.config.ErrorMaxVideoBandwidth != 0 && maxVideoBandwidth > ins.config.ErrorMaxVideoBandwidth {
		return &core.Report{
			Name:     "RepresentationInspector",
			Severity: core.Error,
			Message:  "high video bandwidth",
			Values:   values,
		}
	}
	if ins.config.WarnMaxVideoBandwidth != 0 && maxVideoBandwidth > ins.config.WarnMaxVideoBandwidth {
		return &core.Report{
			Name:     "RepresentationInspector",
			Severity: core.Warn,
			Message:  "high video bandwidth",
			Values:   values,
		}
	}
	if ins.config.ErrorMinVideoBandwidth != 0 && minVideoBandwidth < ins.config.ErrorMinVideoBandwidth {
		return &core.Report{
			Name:     "RepresentationInspector",
			Severity: core.Error,
			Message:  "low video bandwidth",
			Values:   values,
		}
	}
	if ins.config.WarnMinVideoBandwidth != 0 && minVideoBandwidth < ins.config.WarnMinVideoBandwidth {
		return &core.Report{
			Name:     "RepresentationInspector",
			Severity: core.Warn,
			Message:  "low video bandwidth",
			Values:   values,
		}
	}
	if ins.config.ErrorMaxAudioBandwidth != 0 && maxAudioBandwidth > ins.config.ErrorMaxAudioBandwidth {
		return &core.Report{
			Name:     "RepresentationInspector",
			Severity: core.Error,
			Message:  "high audio bandwidth",
			Values:   values,
		}
	}
	if ins.config.WarnMaxAudioBandwidth != 0 && maxAudioBandwidth > ins.config.WarnMaxAudioBandwidth {
		return &core.Report{
			Name:     "RepresentationInspector",
			Severity: core.Warn,
			Message:  "high audio bandwidth",
			Values:   values,
		}
	}
	if ins.config.ErrorMinAudioBandwidth != 0 && minAudioBandwidth < ins.config.ErrorMinAudioBandwidth {
		return &core.Report{
			Name:     "RepresentationInspector",
			Severity: core.Error,
			Message:  "low audio bandwidth",
			Values:   values,
		}
	}
	if ins.config.WarnMinAudioBandwidth != 0 && minAudioBandwidth < ins.config.WarnMinAudioBandwidth {
		return &core.Report{
			Name:     "RepresentationInspector",
			Severity: core.Warn,
			Message:  "low audio bandwidth",
			Values:   values,
		}
	}

	return &core.Report{
		Name:     "RepresentationInspector",
		Severity: core.Info,
		Message:  "good",
		Values:   values,
	}
}

type resolution struct {
	Height *int64
	Width  *int64
	SAR    AspectRatio
}

func getResolution(adaptationSet *mpd.AdaptationSet, representation *mpd.Representation) (*resolution, error) {
	rsl := &resolution{
		SAR: AspectRatio{X: 1, Y: 1},
	}
	if representation.Height != nil {
		rsl.Height = representation.Height
	} else if adaptationSet.Height != nil {
		height, err := strconv.ParseInt(*adaptationSet.Height, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid height: %w", err)
		}
		rsl.Height = &height
	}
	if representation.Width != nil {
		rsl.Width = representation.Width
	} else if adaptationSet.Width != nil {
		width, err := strconv.ParseInt(*adaptationSet.Width, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid width: %w", err)
		}
		rsl.Width = &width
	}
	var sar string
	if representation.Sar != nil {
		sar = *representation.Sar
	} else if adaptationSet.Sar != nil {
		sar = *adaptationSet.Sar
	}
	if sar != "" {
		var err error
		rsl.SAR, err = ParseAspectRatio(sar)
		if err != nil {
			return nil, err
		}
	}
	return rsl, nil
}

func containsAspectRatio(aspectRatio AspectRatio, set []AspectRatio) bool {
	for _, r := range set {
		d := float64(aspectRatio.X)/float64(aspectRatio.Y) - float64(r.X)/float64(r.Y)
		if d < 0.01 && d > -0.01 {
			return true
		}
	}
	return false
}
