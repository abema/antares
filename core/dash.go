package core

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/abema/antares/internal/url"
	"github.com/zencoder/go-dash/mpd"
)

type Manifest struct {
	URL  string
	Raw  []byte
	Time time.Time
	*mpd.MPD
}

type DASHSegment struct {
	URL             string
	Initialization  bool
	Time            uint64
	Duration        uint64
	Period          *mpd.Period
	AdaptationSet   *mpd.AdaptationSet
	SegmentTemplate *mpd.SegmentTemplate
	Representation  *mpd.Representation
}

func (m *Manifest) BaseURL() (string, error) {
	if m.MPD.BaseURL != "" {
		return url.ResolveReference(m.URL, m.MPD.BaseURL)
	}
	return m.URL, nil
}

func (m *Manifest) EachSegments(handle func(*DASHSegment) (cont bool)) error {
	baseURL, err := m.BaseURL()
	if err != nil {
		return err
	}
	for _, period := range m.Periods {
		for _, as := range period.AdaptationSets {
			for _, rep := range as.Representations {
				if as.SegmentTemplate != nil {
					cont, err := visitSegmentsBySegmentTimeline(baseURL, as.SegmentTemplate, period, as, rep, handle)
					if err != nil || !cont {
						return err
					}
				} else if rep.SegmentTemplate != nil {
					cont, err := visitSegmentsBySegmentTimeline(baseURL, rep.SegmentTemplate, period, as, rep, handle)
					if err != nil || !cont {
						return err
					}
				}
			}
		}
	}
	return nil
}

func (m *Manifest) Segments() ([]*DASHSegment, error) {
	segments := make([]*DASHSegment, 0)
	m.EachSegments(func(segment *DASHSegment) bool {
		segments = append(segments, segment)
		return true
	})
	return segments, nil
}

func visitSegmentsBySegmentTimeline(
	baseURL string,
	template *mpd.SegmentTemplate,
	period *mpd.Period,
	as *mpd.AdaptationSet,
	rep *mpd.Representation,
	handle func(*DASHSegment) (cont bool),
) (bool, error) {
	var repID string
	if rep.ID != nil {
		repID = *rep.ID
	}
	var bandwidth int64
	if rep.Bandwidth != nil {
		bandwidth = *rep.Bandwidth
	}
	var number int64
	if template.StartNumber != nil {
		number = *template.StartNumber
	}

	if template.Initialization != nil {
		u, err := url.ResolveReference(baseURL, ResolveTemplate(*template.Initialization, TemplateParams{
			RepresentationID: repID,
			Bandwidth:        bandwidth,
		}))
		if err != nil {
			return false, err
		}
		if !handle(&DASHSegment{
			URL:             u,
			Initialization:  true,
			Period:          period,
			AdaptationSet:   as,
			SegmentTemplate: template,
			Representation:  rep,
		}) {
			return false, nil
		}
	}
	if template.SegmentTimeline != nil && template.Media != nil {
		var t uint64
		for _, segment := range template.SegmentTimeline.Segments {
			n := 1
			if segment.RepeatCount != nil {
				n = *segment.RepeatCount + 1
			}
			if segment.StartTime != nil {
				t = *segment.StartTime
			}
			for i := 0; i < n; i++ {
				u, err := url.ResolveReference(baseURL, ResolveTemplate(*template.Media, TemplateParams{
					RepresentationID: repID,
					Number:           number,
					Bandwidth:        bandwidth,
					Time:             t,
				}))
				if err != nil {
					return false, err
				}
				if !handle(&DASHSegment{
					URL:             u,
					Time:            t,
					Duration:        segment.Duration,
					Period:          period,
					AdaptationSet:   as,
					SegmentTemplate: template,
					Representation:  rep,
				}) {
					return false, nil
				}
				t += segment.Duration
				number++
			}
		}
	}
	return true, nil
}

type TemplateParams struct {
	RepresentationID string
	Number           int64
	Bandwidth        int64
	Time             uint64
}

func ResolveTemplate(format string, params TemplateParams) string {
	var ret string
	ss := strings.Split(format, "$")
	for i, s := range ss {
		if i%2 == 0 {
			ret += s
		} else if s == "" {
			ret += "$"
		} else if s == "RepresentationID" {
			ret += params.RepresentationID
		} else if s == "Number" {
			ret += strconv.FormatInt(params.Number, 10)
		} else if s == "Bandwidth" {
			ret += strconv.FormatInt(params.Bandwidth, 10)
		} else if s == "Time" {
			ret += strconv.FormatUint(params.Time, 10)
		} else {
			ret += "$" + s
			if i == len(ss)-1 {
				ret += "$"
			}
		}
	}
	return ret
}

type dashManifestDownloader struct {
	client   client
	timeout  time.Duration
	location string
}

func newDASHManifestDownloader(client client, timeout time.Duration) *dashManifestDownloader {
	return &dashManifestDownloader{
		client:  client,
		timeout: timeout,
	}
}

func (d *dashManifestDownloader) Download(ctx context.Context, u string) (*Manifest, error) {
	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	if d.location != "" {
		u = d.location
	}
	data, loc, err := d.client.Get(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("failed to download manifest: %s: %w", u, err)
	}
	m, err := mpd.ReadFromString(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode manifest: %s: %w", u, err)
	}
	if m.Location != "" {
		d.location = m.Location
	}
	return &Manifest{
		URL:  loc,
		Raw:  data,
		Time: time.Now(),
		MPD:  m,
	}, nil
}
