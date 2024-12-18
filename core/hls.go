package core

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/abema/antares/internal/thread"
	m3u8 "github.com/abema/go-simple-m3u8"
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
	StreamInfAttrs m3u8.StreamInfAttrs
	MediaAttrs     m3u8.MediaAttrs
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
	// StreamInfAttrs is reference to related StreamInfAttrs object in MasterPlaylist.
	// This property is nullable.
	StreamInfAttrs m3u8.StreamInfAttrs
	// MediaAttrs is reference to related MediaAttrs object in MasterPlaylist.
	// This property is nullable.
	MediaAttrs m3u8.MediaAttrs
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
				URL:            u,
				StreamInfAttrs: playlist.StreamInfAttrs,
				MediaAttrs:     playlist.MediaAttrs,
			})
		}
	}
	return segments, nil
}

func (p *Playlists) IsVOD() bool {
	for _, playlist := range p.MediaPlaylists {
		if !playlist.EndList {
			return false
		}
	}
	return true
}

func (p *Playlists) MaxTargetDuration() float64 {
	var dur float64
	for _, playlist := range p.MediaPlaylists {
		d := float64(playlist.Tags.TargetDuration())
		if d > dur {
			dur = d
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
		dec, err := m3u8.DecodePlaylist(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("failed to decode playlist: %s: %w", u, err)
		}
		if dec.Type() == m3u8.PlaylistTypeMedia {
			media := dec.Media()
			removeNilSegments(media)
			return &Playlists{
				MediaPlaylists: map[string]*MediaPlaylist{
					"_": {
						URL:           loc,
						Raw:           data,
						Time:          time.Now(),
						MediaPlaylist: media,
					},
				},
			}, nil
		}
		master := dec.Master()
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
	for vi := range d.masterPlaylist.Streams {
		variant := d.masterPlaylist.Streams[vi]
		eg.Go(thread.NoPanic(func() error {
			mediaPlaylist, err := d.downloadMediaPlaylist(ctx, base, variant.URI, variant.Attributes, nil)
			if err != nil {
				return err
			}
			mutex.Lock()
			defer mutex.Unlock()
			playlists.MediaPlaylists[variant.URI] = mediaPlaylist
			return nil
		}))
	}
	var alternatives []m3u8.MediaAttrs
	for _, alts := range d.masterPlaylist.Alternatives.Video {
		for _, alt := range alts {
			alternatives = append(alternatives, alt.Attributes)
		}
	}
	for _, alts := range d.masterPlaylist.Alternatives.Audio {
		for _, alt := range alts {
			alternatives = append(alternatives, alt.Attributes)
		}
	}
	for _, alts := range d.masterPlaylist.Alternatives.Subtitles {
		for _, alt := range alts {
			alternatives = append(alternatives, alt.Attributes)
		}
	}
	for _, alt := range alternatives {
		eg.Go(thread.NoPanic(func() error {
			uri := alt.URI()
			mediaPlaylist, err := d.downloadMediaPlaylist(ctx, base, uri, nil, alt)
			if err != nil {
				return err
			}
			mutex.Lock()
			defer mutex.Unlock()
			playlists.MediaPlaylists[uri] = mediaPlaylist
			return nil
		}))
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
	variantParams m3u8.StreamInfAttrs,
	alt m3u8.MediaAttrs,
) (*MediaPlaylist, error) {
	absolute, err := base.Parse(u)
	if err != nil {
		return nil, fmt.Errorf("invalid URL format: %s: %w", u, err)
	}
	data, loc, err := d.client.Get(ctx, absolute.String())
	if err != nil {
		return nil, fmt.Errorf("failed to download media playlist: %s: %w", u, err)
	}
	dec, err := m3u8.DecodePlaylist(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode media playlist: %s: %w", u, err)
	}
	media := dec.Media()
	removeNilSegments(media)
	return &MediaPlaylist{
		URL:            loc,
		Raw:            data,
		Time:           time.Now(),
		MediaPlaylist:  media,
		StreamInfAttrs: variantParams,
		MediaAttrs:     alt,
	}, nil
}

// removeNilSegments removes nil elements, because abema/go-simple-m3u8 returns nil-filled large slice.
// https://github.com/abema/go-simple-m3u8/issues/97
func removeNilSegments(media *m3u8.MediaPlaylist) {
	for i := range media.Segments {
		if media.Segments[i] == nil {
			media.Segments = media.Segments[:i]
			break
		}
	}
}
