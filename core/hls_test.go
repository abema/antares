package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"github.com/grafov/m3u8"
	"github.com/stretchr/testify/require"
)

func TestSegmentURLs(t *testing.T) {
	p := &Playlists{
		MediaPlaylists: map[string]*MediaPlaylist{
			"media_0.m3u8": {
				URL: "https://localhost/foo/media_0.m3u8",
				MediaPlaylist: &m3u8.MediaPlaylist{Segments: []*m3u8.MediaSegment{
					{URI: "segment_0_0.ts"},
					{URI: "segment_0_1.ts"},
				}},
				VariantParams: &m3u8.VariantParams{Bandwidth: uint32(2000000)},
				Alternative:   &m3u8.Alternative{Name: "audio_0"},
			},
			"media_1.m3u8": {
				URL: "https://localhost/foo/media_1.m3u8",
				MediaPlaylist: &m3u8.MediaPlaylist{Segments: []*m3u8.MediaSegment{
					{URI: "segment_1_0.ts"},
					{URI: "segment_1_1.ts"},
				}},
				VariantParams: &m3u8.VariantParams{Bandwidth: uint32(1000000)},
				Alternative:   &m3u8.Alternative{Name: "audio_1"},
			},
		},
	}
	segments, err := p.Segments()
	require.NoError(t, err)
	require.Len(t, segments, 4)
	sort.Slice(segments, func(i, j int) bool {
		return segments[i].URL < segments[j].URL
	})
	require.Equal(t, "https://localhost/foo/segment_0_0.ts", segments[0].URL)
	require.Equal(t, "https://localhost/foo/segment_0_1.ts", segments[1].URL)
	require.Equal(t, "https://localhost/foo/segment_1_0.ts", segments[2].URL)
	require.Equal(t, "https://localhost/foo/segment_1_1.ts", segments[3].URL)
	require.Equal(t, uint32(2000000), segments[0].VariantParams.Bandwidth)
	require.Equal(t, "audio_0", segments[0].Alternative.Name)
	require.Equal(t, uint32(1000000), segments[2].VariantParams.Bandwidth)
	require.Equal(t, "audio_1", segments[2].Alternative.Name)
}

func TestIsVOD(t *testing.T) {
	t.Run("all_live", func(t *testing.T) {
		p := &Playlists{
			MediaPlaylists: map[string]*MediaPlaylist{
				"media_0.m3u8": {
					MediaPlaylist: &m3u8.MediaPlaylist{Closed: false},
				},
				"media_1.m3u8": {
					MediaPlaylist: &m3u8.MediaPlaylist{Closed: false},
				},
			},
		}
		require.False(t, p.IsVOD())
	})

	t.Run("vod_and_live", func(t *testing.T) {
		p := &Playlists{
			MediaPlaylists: map[string]*MediaPlaylist{
				"media_0.m3u8": {
					MediaPlaylist: &m3u8.MediaPlaylist{Closed: false},
				},
				"media_1.m3u8": {
					MediaPlaylist: &m3u8.MediaPlaylist{Closed: true},
				},
			},
		}
		require.False(t, p.IsVOD())
	})

	t.Run("vod", func(t *testing.T) {
		p := &Playlists{
			MediaPlaylists: map[string]*MediaPlaylist{
				"media_0.m3u8": {
					MediaPlaylist: &m3u8.MediaPlaylist{Closed: true},
				},
				"media_1.m3u8": {
					MediaPlaylist: &m3u8.MediaPlaylist{Closed: true},
				},
			},
		}
		require.True(t, p.IsVOD())
	})
}

func TestMaxTargetDuration(t *testing.T) {
	p := &Playlists{
		MediaPlaylists: map[string]*MediaPlaylist{
			"media_0.m3u8": {
				MediaPlaylist: &m3u8.MediaPlaylist{TargetDuration: 6.0},
			},
			"media_1.m3u8": {
				MediaPlaylist: &m3u8.MediaPlaylist{TargetDuration: 5.0},
			},
		},
	}
	require.Equal(t, 6.0, p.MaxTargetDuration())
}

func TestHLSPlaylistDownloader(t *testing.T) {
	master := []byte(`#EXTM3U` + "\n" +
		`#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="audio",DEFAULT=YES,URI="media_2.m3u8"` + "\n" +
		`#EXT-X-STREAM-INF:BANDWIDTH=1280000,AVERAGE-BANDWIDTH=1000000,AUDIO="audio"` + "\n" +
		`media_0.m3u8` + "\n" +
		`#EXT-X-STREAM-INF:BANDWIDTH=2560000,AVERAGE-BANDWIDTH=2000000,AUDIO="audio"` + "\n" +
		`media_1.m3u8` + "\n")
	media0 := []byte(`#EXTM3U` + "\n" +
		`#EXT-X-VERSION:3` + "\n" +
		`#EXT-X-TARGETDURATION:8` + "\n" +
		`#EXT-X-MEDIA-SEQUENCE:2680` + "\n" +
		`#EXTINF:7.975,` + "\n" +
		`media_0_100.ts` + "\n" +
		`#EXTINF:7.941,` + "\n" +
		`media_0_101.ts` + "\n")
	media1 := []byte(`#EXTM3U` + "\n" +
		`#EXT-X-VERSION:3` + "\n" +
		`#EXT-X-TARGETDURATION:8` + "\n" +
		`#EXT-X-MEDIA-SEQUENCE:2680` + "\n" +
		`#EXTINF:7.975,` + "\n" +
		`media_1_100.ts` + "\n" +
		`#EXTINF:7.941,` + "\n" +
		`media_1_101.ts` + "\n")
	media2 := []byte(`#EXTM3U` + "\n" +
		`#EXT-X-VERSION:3` + "\n" +
		`#EXT-X-TARGETDURATION:8` + "\n" +
		`#EXT-X-MEDIA-SEQUENCE:2680` + "\n" +
		`#EXTINF:7.975,` + "\n" +
		`media_2_100.ts` + "\n" +
		`#EXTINF:7.941,` + "\n" +
		`media_2_101.ts` + "\n")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/master.m3u8":
			w.Write(master)
		case "/media_0.m3u8":
			w.Write(media0)
		case "/media_1.m3u8":
			w.Write(media1)
		case "/media_2.m3u8":
			w.Write(media2)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	t.Run("master playlist", func(t *testing.T) {
		d := newHLSPlaylistDownloader(newClient(http.DefaultClient, nil, nil), time.Second)
		playlists, err := d.Download(context.Background(), server.URL+"/master.m3u8")
		require.NoError(t, err)
		require.Equal(t, server.URL+"/master.m3u8", playlists.MasterPlaylist.URL)
		require.Equal(t, master, playlists.MasterPlaylist.Raw)
		require.Len(t, playlists.MasterPlaylist.Variants, 2)
		require.Equal(t, "media_0.m3u8", playlists.MasterPlaylist.Variants[0].URI)
		require.Equal(t, "media_1.m3u8", playlists.MasterPlaylist.Variants[1].URI)
		require.Len(t, playlists.MediaPlaylists, 3)

		mp0 := playlists.MediaPlaylists["media_0.m3u8"]
		require.Equal(t, server.URL+"/media_0.m3u8", mp0.URL)
		require.Equal(t, media0, mp0.Raw)
		require.Len(t, mp0.Segments, 2)
		require.Equal(t, "media_0_100.ts", mp0.Segments[0].URI)
		require.Equal(t, "media_0_101.ts", mp0.Segments[1].URI)
		require.Equal(t, uint32(1280000), mp0.VariantParams.Bandwidth)
		require.Equal(t, uint32(1000000), mp0.VariantParams.AverageBandwidth)
		require.Nil(t, mp0.Alternative)

		mp1 := playlists.MediaPlaylists["media_1.m3u8"]
		require.Equal(t, server.URL+"/media_1.m3u8", mp1.URL)
		require.Equal(t, media1, mp1.Raw)
		require.Len(t, mp1.Segments, 2)
		require.Equal(t, "media_1_100.ts", mp1.Segments[0].URI)
		require.Equal(t, "media_1_101.ts", mp1.Segments[1].URI)
		require.Equal(t, uint32(2560000), mp1.VariantParams.Bandwidth)
		require.Equal(t, uint32(2000000), mp1.VariantParams.AverageBandwidth)
		require.Nil(t, mp1.Alternative)

		mp2 := playlists.MediaPlaylists["media_2.m3u8"]
		require.Equal(t, server.URL+"/media_2.m3u8", mp2.URL)
		require.Equal(t, media2, mp2.Raw)
		require.Len(t, mp2.Segments, 2)
		require.Equal(t, "media_2_100.ts", mp2.Segments[0].URI)
		require.Equal(t, "media_2_101.ts", mp2.Segments[1].URI)
		require.Nil(t, mp2.VariantParams)
		require.Equal(t, "audio", mp2.Alternative.GroupId)
	})

	t.Run("single media playlist", func(t *testing.T) {
		d := newHLSPlaylistDownloader(newClient(http.DefaultClient, nil, nil), time.Second)
		playlists, err := d.Download(context.Background(), server.URL+"/media_0.m3u8")
		require.NoError(t, err)
		require.Nil(t, playlists.MasterPlaylist)
		require.Len(t, playlists.MediaPlaylists, 1)

		mp0 := playlists.MediaPlaylists["_"]
		require.Equal(t, server.URL+"/media_0.m3u8", mp0.URL)
		require.Equal(t, media0, mp0.Raw)
		require.Len(t, mp0.Segments, 2)
		require.Equal(t, "media_0_100.ts", mp0.Segments[0].URI)
		require.Equal(t, "media_0_101.ts", mp0.Segments[1].URI)
		require.Nil(t, mp0.VariantParams)
		require.Nil(t, mp0.Alternative)
	})
}
