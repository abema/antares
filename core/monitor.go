package core

import (
	"context"
	"errors"
	"log"
	"sort"
	"time"

	"github.com/abema/antares/internal/thread"
	backoff "github.com/cenkalti/backoff/v4"
	"github.com/zencoder/go-dash/mpd"
)

type Monitor interface {
	Terminate()
}

type monitor struct {
	config         *Config
	httpClient     client
	hlsDownloader  *hlsPlaylistDownloader
	dashDownloader *dashManifestDownloader
	segmentStore   mutableSegmentStore
	context        context.Context
	terminate      func()
}

func NewMonitor(config *Config) Monitor {
	httpClient := newClient(config.HTTPClient, config.RequestHeader, config.OnDownload)
	m := &monitor{
		config:       config,
		httpClient:   httpClient,
		segmentStore: newSegmentStore(httpClient, config.SegmentTimeout, config.SegmentMaxConcurrency),
	}
	manifestClient := httpClient
	if !config.NoRedirectCache {
		manifestClient = newRedirectKeeper(manifestClient)
	}
	switch config.StreamType {
	case StreamTypeHLS:
		m.hlsDownloader = newHLSPlaylistDownloader(manifestClient, config.ManifestTimeout)
	case StreamTypeDASH:
		m.dashDownloader = newDASHManifestDownloader(manifestClient, config.ManifestTimeout)
	}
	m.context, m.terminate = context.WithCancel(context.Background())
	go m.run()
	return m
}

func (m *monitor) Terminate() {
	m.terminate()
}

func (m *monitor) run() {
	for {
		cont, waitDur := m.proc()
		if !cont {
			break
		}
		if cont := m.wait(waitDur); !cont {
			break
		}
	}
	if m.config.OnTerminate != nil {
		m.config.OnTerminate()
	}
}

func (m *monitor) proc() (cont bool, waitDur time.Duration) {
	defer func() {
		if r := recover(); r != nil {
			m.onPanic(r)
			cont = true
			waitDur = m.config.DefaultInterval
		}
	}()

	var playlists *Playlists
	var manifest *Manifest
	err := backoff.RetryNotify(func() error {
		var err error
		switch m.config.StreamType {
		case StreamTypeHLS:
			playlists, err = m.hlsDownloader.Download(m.context, m.config.URL)
		default:
			manifest, err = m.dashDownloader.Download(m.context, m.config.URL)
		}
		if err != nil && (m.context.Err() != nil || errors.As(err, &permanentError{})) {
			return backoff.Permanent(err)
		}
		return err
	}, m.config.ManifestBackoff, func(err error, _ time.Duration) {
		log.Printf("WARN: failed to download manifest: %s: %s", m.config.URL, err)
	})
	if err != nil {
		m.onError("failed to download manifest", err)
		return true, m.config.DefaultInterval
	}

	switch m.config.StreamType {
	case StreamTypeHLS:
		if err := m.updateSegmentStoreHLS(playlists); err != nil {
			m.onError("failed to download segment", err)
			return true, m.config.DefaultInterval
		}
	default:
		if err := m.updateSegmentStoreDASH(manifest); err != nil {
			m.onError("failed to download segment", err)
			return true, m.config.DefaultInterval
		}
	}

	rc := make(chan *Report)
	var ni int
	switch m.config.StreamType {
	case StreamTypeHLS:
		for _, inspector := range m.config.HLS.Inspectors {
			go func(inspector HLSInspector) {
				rc <- inspector.Inspect(playlists, m.segmentStore)
			}(inspector)
			ni++
		}
	default:
		for _, inspector := range m.config.DASH.Inspectors {
			go func(inspector DASHInspector) {
				rc <- inspector.Inspect(manifest, m.segmentStore)
			}(inspector)
			ni++
		}
	}
	reports := make([]*Report, 0, ni)
	for i := 0; i < ni; i++ {
		if rep := <-rc; rep != nil {
			reports = append(reports, rep)
		}
	}
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].Name < reports[j].Name
	})
	m.onReport(reports)

	switch m.config.StreamType {
	case StreamTypeHLS:
		return m.hlsWaitDuration(playlists)
	default:
		return m.dashWaitDuration(manifest)
	}
}

func (m *monitor) hlsWaitDuration(playlists *Playlists) (bool, time.Duration) {
	if playlists.IsVOD() {
		if m.config.TerminateIfVOD {
			return false, 0
		}
		return true, m.config.DefaultInterval
	}
	if !m.config.PrioritizeSuggestedInterval {
		return true, m.config.DefaultInterval
	}
	dur := time.Duration(playlists.MaxTargetDuration()) * time.Second / 2
	if dur == 0 {
		return true, m.config.DefaultInterval
	} else if dur < time.Second {
		return true, time.Second
	}
	return true, dur
}

func (m *monitor) dashWaitDuration(manifest *Manifest) (bool, time.Duration) {
	if manifest.Type == nil || *manifest.Type != "dynamic" {
		if m.config.TerminateIfVOD {
			return false, 0
		}
		return true, m.config.DefaultInterval
	}
	if !m.config.PrioritizeSuggestedInterval || manifest.MinimumUpdatePeriod == nil {
		return true, m.config.DefaultInterval
	}
	dur, err := mpd.ParseDuration(*manifest.MinimumUpdatePeriod)
	if err != nil {
		log.Printf("ERROR: failed to parse minimumUpdatePeriod: %s: %s", manifest.URL, err)
		return true, m.config.DefaultInterval
	} else if dur < time.Second {
		return true, time.Second
	}
	return true, dur
}

func (m *monitor) wait(dur time.Duration) bool {
	select {
	case <-time.After(dur):
		return true
	case <-m.context.Done():
		return false
	}
}

func (m *monitor) updateSegmentStoreHLS(playlists *Playlists) error {
	segments, err := playlists.Segments()
	if err != nil {
		return err
	}
	urls := make([]string, 0, len(segments))
	for _, seg := range segments {
		if m.config.SegmentFilter == nil || m.config.SegmentFilter.CheckHLS(seg) == Pass {
			urls = append(urls, seg.URL)
		}
	}
	return m.segmentStore.Sync(m.context, urls)
}

func (m *monitor) updateSegmentStoreDASH(manifest *Manifest) error {
	segments, err := manifest.Segments()
	if err != nil {
		return err
	}
	urls := make([]string, 0, len(segments))
	for _, seg := range segments {
		if m.config.SegmentFilter == nil || m.config.SegmentFilter.CheckDASH(seg) == Pass {
			urls = append(urls, seg.URL)
		}
	}
	return m.segmentStore.Sync(m.context, urls)
}

func (m *monitor) onPanic(r interface{}) {
	m.onError("panic is occurred", thread.PanicToError(r, nil))
}

func (m *monitor) onError(msg string, err error) {
	m.onReport([]*Report{
		{
			Name:     "Monitor",
			Severity: Error,
			Message:  msg,
			Values: Values{
				"error": err,
			},
		},
	})
}

func (m *monitor) onReport(reports Reports) {
	if m.config.OnReport != nil {
		m.config.OnReport(reports)
	}
}
