package core

import (
	"net/http"
	"time"

	backoff "github.com/cenkalti/backoff/v4"
)

type StreamType int

const (
	StreamTypeHLS StreamType = iota
	StreamTypeDASH
)

func (t StreamType) String() string {
	switch t {
	case StreamTypeHLS:
		return "HLS"
	case StreamTypeDASH:
		return "DASH"
	}
	return "<Unknown>"
}

type HLSConfig struct {
	Inspectors []HLSInspector
}

type DASHConfig struct {
	Inspectors []DASHInspector
}

type Config struct {
	URL                         string
	DefaultInterval             time.Duration
	PrioritizeSuggestedInterval bool
	HTTPClient                  *http.Client
	RequestHeader               http.Header
	NoRedirectCache             bool
	ManifestTimeout             time.Duration
	ManifestBackoff             backoff.BackOff
	SegmentTimeout              time.Duration
	SegmentBackoff              backoff.BackOff
	SegmentMaxConcurrency       int
	SegmentFilter               SegmentFilter
	StreamType                  StreamType
	TerminateIfVOD              bool
	HLS                         *HLSConfig
	DASH                        *DASHConfig
	// OnDownload will be called when HTTP GET method succeeds.
	// This function must be thread-safe.
	OnDownload  OnDownloadHandler
	OnReport    OnReportHandler
	OnTerminate OnTerminateHandler
}

func NewConfig(url string, streamType StreamType) *Config {
	backoff := backoff.NewExponentialBackOff()
	backoff.MaxInterval = 2 * time.Second
	backoff.MaxElapsedTime = 10 * time.Second
	config := &Config{
		URL:                   url,
		DefaultInterval:       5 * time.Second,
		HTTPClient:            http.DefaultClient,
		ManifestTimeout:       1 * time.Second,
		ManifestBackoff:       backoff,
		SegmentTimeout:        3 * time.Second,
		SegmentBackoff:        backoff,
		SegmentMaxConcurrency: 4,
		StreamType:            streamType,
	}
	switch streamType {
	case StreamTypeHLS:
		config.HLS = &HLSConfig{}
	case StreamTypeDASH:
		config.DASH = &DASHConfig{}
	}
	return config
}
