package core

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/abema/antares/internal/thread"
	"github.com/grafov/m3u8"
	"golang.org/x/sync/errgroup"
)

type MasterPlaylist struct {
	URL  string
	Raw  []byte
	Time time.Time
	*m3u8.MasterPlaylist
}

type MediaPlaylist struct {
	URL  string
	Raw  []byte
	Time time.Time
	*m3u8.MediaPlaylist
	VariantParams *m3u8.VariantParams
	Alternative   *m3u8.Alternative
}

func (p *MediaPlaylist) SegmentURLs() ([]string, error) {
	base, err := url.Parse(p.URL)
	if err != nil {
		return nil, err
	}
	urls := make([]string, 0)
	for _, segment := range p.Segments {
		u, err := base.Parse(segment.URI)
		if err != nil {
			return nil, err
		}
		urls = append(urls, u.String())
	}
	return urls, nil
}

type Playlists struct {
	MasterPlaylist *MasterPlaylist
	MediaPlaylists map[string]*MediaPlaylist
}

type HLSSegment struct {
	URL string
	// VariantParams is reference to related VariantParams object in MasterPlaylist.
	// This property is nullable.
	VariantParams *m3u8.VariantParams
	// Alternative is reference to related Alternative object in MasterPlaylist.
	// This property is nullable.
	Alternative *m3u8.Alternative
}

func (p *Playlists) Segments() ([]*HLSSegment, error) {
	segments := make([]*HLSSegment, 0)
	for _, playlist := range p.MediaPlaylists {
		urls, err := playlist.SegmentURLs()
		if err != nil {
			return nil, err
		}
		for _, u := range urls {
			segments = append(segments, &HLSSegment{
				URL:           u,
				VariantParams: playlist.VariantParams,
				Alternative:   playlist.Alternative,
			})
		}
	}
	return segments, nil
}

func (p *Playlists) IsVOD() bool {
	for _, playlist := range p.MediaPlaylists {
		if !playlist.Closed {
			return false
		}
	}
	return true
}

func (p *Playlists) MaxTargetDuration() float64 {
	var dur float64
	for _, playlist := range p.MediaPlaylists {
		if playlist.TargetDuration > dur {
			dur = playlist.TargetDuration
		}
	}
	return dur
}

type hlsPlaylistDownloader struct {
	client         client
	timeout        time.Duration
	masterPlaylist *MasterPlaylist
}

func newHLSPlaylistDownloader(client client, timeout time.Duration) *hlsPlaylistDownloader {
	return &hlsPlaylistDownloader{
		client:  client,
		timeout: timeout,
	}
}

func (d *hlsPlaylistDownloader) Download(ctx context.Context, u string) (*Playlists, error) {
	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	if d.masterPlaylist == nil {
		data, loc, err := d.client.Get(ctx, u)
		if err != nil {
			return nil, fmt.Errorf("failed to download playlist: %s: %w", u, err)
		}
		dec, ptype, err := m3u8.DecodeFrom(bytes.NewReader(data), true)
		if err != nil {
			return nil, fmt.Errorf("failed to decode playlist: %s: %w", u, err)
		}
		if ptype == m3u8.MEDIA {
			media := dec.(*m3u8.MediaPlaylist)
			removeNilSegments(media)
			return &Playlists{
				MediaPlaylists: map[string]*MediaPlaylist{
					"_": &MediaPlaylist{
						URL:           loc,
						Raw:           data,
						Time:          time.Now(),
						MediaPlaylist: media,
					},
				},
			}, nil
		}
		master := dec.(*m3u8.MasterPlaylist)
		d.masterPlaylist = &MasterPlaylist{
			URL:            loc,
			Raw:            data,
			Time:           time.Now(),
			MasterPlaylist: master,
		}
	}

	playlists := &Playlists{
		MasterPlaylist: d.masterPlaylist,
		MediaPlaylists: make(map[string]*MediaPlaylist),
	}
	base, err := url.Parse(u)
	if err != nil {
		return nil, err
	}
	var mutex sync.Mutex
	eg := new(errgroup.Group)
	for vi := range d.masterPlaylist.Variants {
		variant := d.masterPlaylist.Variants[vi]
		eg.Go(thread.NoPanic(func() error {
			mediaPlaylist, err := d.downloadMediaPlaylist(ctx, base, variant.URI, &variant.VariantParams, nil)
			if err != nil {
				return err
			}
			mutex.Lock()
			defer mutex.Unlock()
			playlists.MediaPlaylists[variant.URI] = mediaPlaylist
			return nil
		}))
		for ai := range variant.Alternatives {
			alt := variant.Alternatives[ai]
			if _, exists := playlists.MediaPlaylists[alt.URI]; exists {
				continue
			}
			eg.Go(thread.NoPanic(func() error {
				mediaPlaylist, err := d.downloadMediaPlaylist(ctx, base, alt.URI, nil, alt)
				if err != nil {
					return err
				}
				mutex.Lock()
				defer mutex.Unlock()
				playlists.MediaPlaylists[alt.URI] = mediaPlaylist
				return nil
			}))
		}
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return playlists, nil
}

func (d *hlsPlaylistDownloader) downloadMediaPlaylist(
	ctx context.Context,
	base *url.URL,
	u string,
	variantParams *m3u8.VariantParams,
	alt *m3u8.Alternative,
) (*MediaPlaylist, error) {
	absolute, err := base.Parse(u)
	if err != nil {
		return nil, fmt.Errorf("invalid URL format: %s: %w", u, err)
	}
	data, loc, err := d.client.Get(ctx, absolute.String())
	if err != nil {
		return nil, fmt.Errorf("failed to download media playlist: %s: %w", u, err)
	}
	dec, _, err := m3u8.DecodeFrom(bytes.NewReader(data), true)
	if err != nil {
		return nil, fmt.Errorf("failed to decode media playlist: %s: %w", u, err)
	}
	media := dec.(*m3u8.MediaPlaylist)
	removeNilSegments(media)
	return &MediaPlaylist{
		URL:           loc,
		Raw:           data,
		Time:          time.Now(),
		MediaPlaylist: media,
		VariantParams: variantParams,
		Alternative:   alt,
	}, nil
}

// removeNilSegments removes nil elements, because grafov/m3u8 returns nil-filled large slice.
// https://github.com/grafov/m3u8/issues/97
func removeNilSegments(media *m3u8.MediaPlaylist) {
	for i := range media.Segments {
		if media.Segments[i] == nil {
			media.Segments = media.Segments[:i]
			break
		}
	}
}
