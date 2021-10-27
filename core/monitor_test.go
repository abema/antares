package core

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/grafov/m3u8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zencoder/go-dash/helpers/ptrs"
	"github.com/zencoder/go-dash/mpd"
)

type mockHLSInspector struct {
	inspect func(playlists *Playlists, segments SegmentStore) *Report
}

func (ins *mockHLSInspector) Inspect(playlists *Playlists, segments SegmentStore) *Report {
	return ins.inspect(playlists, segments)
}

type mockDASHInspector struct {
	inspect func(manifest *Manifest, segments SegmentStore) *Report
}

func (ins *mockDASHInspector) Inspect(manifest *Manifest, segments SegmentStore) *Report {
	return ins.inspect(manifest, segments)
}

func TestMonitor(t *testing.T) {
	t.Run("HLS Live", func(t *testing.T) {
		master := []byte(`#EXTM3U` + "\n" +
			`#EXT-X-STREAM-INF:BANDWIDTH=1280000,AVERAGE-BANDWIDTH=1000000` + "\n" +
			`media_0.m3u8` + "\n" +
			`#EXT-X-STREAM-INF:BANDWIDTH=2560000,AVERAGE-BANDWIDTH=2000000` + "\n" +
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
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/master.m3u8":
				w.Write(master)
			case "/media_0.m3u8":
				w.Write(media0)
			case "/media_1.m3u8":
				w.Write(media1)
			case "/media_0_100.ts", "/media_0_101.ts",
				"/media_1_100.ts", "/media_1_101.ts":
				w.Write([]byte("dummy TS file:" + r.URL.Path))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		config := NewConfig(server.URL+"/master.m3u8", StreamTypeHLS)
		config.HLS.Inspectors = []HLSInspector{
			&mockHLSInspector{inspect: func(playlists *Playlists, segments SegmentStore) *Report {
				ts, ok := segments.Load(server.URL + "/media_0_100.ts")
				require.True(t, ok)
				assert.Equal(t, "dummy TS file:/media_0_100.ts", string(ts))
				assert.True(t, segments.Exists(server.URL+"/media_0_101.ts"))
				assert.False(t, segments.Exists(server.URL+"/media_0_102.ts"))
				return &Report{Name: "i1", Severity: Info}
			}},
			&mockHLSInspector{inspect: func(playlists *Playlists, segments SegmentStore) *Report {
				return &Report{Name: "i2", Severity: Warn}
			}},
		}
		callCh := make(chan string, 100)
		config.OnDownload = func(file *File) {
			switch file.URL {
			case server.URL + "/master.m3u8":
				assert.Equal(t, master, file.Body)
			case server.URL + "/media_0.m3u8":
				assert.Equal(t, media0, file.Body)
			case server.URL + "/media_1.m3u8":
				assert.Equal(t, media1, file.Body)
			case server.URL + "/media_0_100.ts":
				assert.Equal(t, []byte("dummy TS file:/media_0_100.ts"), file.Body)
			case server.URL + "/media_0_101.ts":
				assert.Equal(t, []byte("dummy TS file:/media_0_101.ts"), file.Body)
			case server.URL + "/media_1_100.ts":
				assert.Equal(t, []byte("dummy TS file:/media_1_100.ts"), file.Body)
			case server.URL + "/media_1_101.ts":
				assert.Equal(t, []byte("dummy TS file:/media_1_101.ts"), file.Body)
			default:
				require.Fail(t, "unexpected URL", file.URL)
			}
			callCh <- "OnDownload"
		}
		config.OnReport = func(reports Reports) {
			assert.Len(t, reports, 2)
			assert.Equal(t, Info, reports[0].Severity)
			assert.Equal(t, Warn, reports[1].Severity)
			callCh <- "OnReport"
		}
		config.OnTerminate = func() {
			callCh <- "OnTerminate"
		}

		m := NewMonitor(config)
		time.Sleep(500 * time.Millisecond)
		m.Terminate()
		time.Sleep(100 * time.Millisecond)
		close(callCh)

		calls := make([]string, 0)
		for call := range callCh {
			calls = append(calls, call)
		}
		require.Len(t, calls, 9)
		for i := 0; i < 7; i++ {
			assert.Equal(t, "OnDownload", calls[i])
		}
		assert.Equal(t, "OnReport", calls[7])
		assert.Equal(t, "OnTerminate", calls[8])
	})

	t.Run("DASH Live", func(t *testing.T) {
		manifest := []byte(`<MPD type="dynamic" minimumUpdatePeriod="PT5.000000S" availabilityStartTime="1970-01-01T00:00:00Z">` +
			`<Period id="1" start="PT1600000000.000S">` +
			`<AdaptationSet mimeType="video/mp4">` +
			`<SegmentTemplate timescale="90000" media="media_$RepresentationID$_$Time$.mp4" initialization="media_$RepresentationID$_init.mp4">` +
			`<SegmentTimeline>` +
			`<S d="900000" t="0"/><S d="900000"/>` +
			`</SegmentTimeline>` +
			`</SegmentTemplate>` +
			`<Representation id="4M" bandwidth="4000000" codecs="avc1.4D401F" frameRate="30000/1001" height="1080" width="1920" />` +
			`<Representation id="2M" bandwidth="2000000" codecs="avc1.4D401F" frameRate="30000/1001" height="720" width="1280" />` +
			`</AdaptationSet>` +
			`</Period>` +
			`</MPD>`)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/manifest.mpd":
				w.Write(manifest)
			case "/media_4M_init.mp4", "/media_4M_0.mp4", "/media_4M_900000.mp4",
				"/media_2M_init.mp4", "/media_2M_0.mp4", "/media_2M_900000.mp4":
				w.Write([]byte("dummy MP4 file:" + r.URL.Path))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		config := NewConfig(server.URL+"/manifest.mpd", StreamTypeDASH)
		config.DASH.Inspectors = []DASHInspector{
			&mockDASHInspector{inspect: func(manifest *Manifest, segments SegmentStore) *Report {
				mp4, ok := segments.Load(server.URL + "/media_4M_init.mp4")
				require.True(t, ok)
				assert.Equal(t, "dummy MP4 file:/media_4M_init.mp4", string(mp4))
				assert.True(t, segments.Exists(server.URL+"/media_4M_0.mp4"))
				assert.True(t, segments.Exists(server.URL+"/media_4M_900000.mp4"))
				assert.False(t, segments.Exists(server.URL+"/media_4M_1800000.mp4"))
				return &Report{Name: "i1", Severity: Info}
			}},
			&mockDASHInspector{inspect: func(manifest *Manifest, segments SegmentStore) *Report {
				return &Report{Name: "i2", Severity: Warn}
			}},
		}
		callCh := make(chan string, 100)
		config.OnDownload = func(file *File) {
			switch file.URL {
			case server.URL + "/manifest.mpd":
				assert.Equal(t, manifest, file.Body)
			case server.URL + "/media_4M_init.mp4":
				assert.Equal(t, []byte("dummy MP4 file:/media_4M_init.mp4"), file.Body)
			case server.URL + "/media_4M_0.mp4":
				assert.Equal(t, []byte("dummy MP4 file:/media_4M_0.mp4"), file.Body)
			case server.URL + "/media_4M_900000.mp4":
				assert.Equal(t, []byte("dummy MP4 file:/media_4M_900000.mp4"), file.Body)
			case server.URL + "/media_2M_init.mp4":
				assert.Equal(t, []byte("dummy MP4 file:/media_2M_init.mp4"), file.Body)
			case server.URL + "/media_2M_0.mp4":
				assert.Equal(t, []byte("dummy MP4 file:/media_2M_0.mp4"), file.Body)
			case server.URL + "/media_2M_900000.mp4":
				assert.Equal(t, []byte("dummy MP4 file:/media_2M_900000.mp4"), file.Body)
			default:
				require.Fail(t, "unexpected URL", file.URL)
			}
			callCh <- "OnDownload"
		}
		config.OnReport = func(reports Reports) {
			assert.Len(t, reports, 2)
			assert.Equal(t, Info, reports[0].Severity)
			assert.Equal(t, Warn, reports[1].Severity)
			callCh <- "OnReport"
		}
		config.OnTerminate = func() {
			callCh <- "OnTerminate"
		}

		m := NewMonitor(config)
		time.Sleep(500 * time.Millisecond)
		m.Terminate()
		time.Sleep(100 * time.Millisecond)
		close(callCh)

		calls := make([]string, 0)
		for call := range callCh {
			calls = append(calls, call)
		}
		require.Len(t, calls, 9)
		for i := 0; i < 7; i++ {
			assert.Equal(t, "OnDownload", calls[i])
		}
		assert.Equal(t, "OnReport", calls[7])
		assert.Equal(t, "OnTerminate", calls[8])
	})
}

func TestMonitor_HLSWaitDuration(t *testing.T) {
	livePlaylists := &Playlists{MediaPlaylists: map[string]*MediaPlaylist{
		"1.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{
			TargetDuration: 8.0,
			Closed:         false,
		}},
	}}
	vodPlaylists := &Playlists{MediaPlaylists: map[string]*MediaPlaylist{
		"1.m3u8": {MediaPlaylist: &m3u8.MediaPlaylist{
			TargetDuration: 8.0,
			Closed:         true,
		}},
	}}

	testCases := []struct {
		name      string
		playlists *Playlists
		config    *Config
		cont      bool
		dur       time.Duration
	}{
		{
			name:      "live_default",
			playlists: livePlaylists,
			config: &Config{
				DefaultInterval:             5 * time.Second,
				PrioritizeSuggestedInterval: false,
			},
			cont: true,
			dur:  5 * time.Second,
		},
		{
			name:      "live_suggested",
			playlists: livePlaylists,
			config: &Config{
				DefaultInterval:             5 * time.Second,
				PrioritizeSuggestedInterval: true,
			},
			cont: true,
			dur:  4 * time.Second,
		},
		{
			name:      "vod_not_terminate",
			playlists: vodPlaylists,
			config: &Config{
				DefaultInterval:             5 * time.Second,
				PrioritizeSuggestedInterval: true,
			},
			cont: true,
			dur:  5 * time.Second,
		},
		{
			name:      "vod_terminate",
			playlists: vodPlaylists,
			config: &Config{
				DefaultInterval:             5 * time.Second,
				PrioritizeSuggestedInterval: true,
				TerminateIfVOD:              true,
			},
			cont: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewMonitor(tc.config).(*monitor)
			cont, dur := m.hlsWaitDuration(tc.playlists)
			assert.Equal(t, tc.cont, cont)
			assert.Equal(t, tc.dur, dur)
		})
	}
}

func TestMonitor_DASHWaitDuration(t *testing.T) {
	liveManifest := &Manifest{
		MPD: &mpd.MPD{
			Type:                ptrs.Strptr("dynamic"),
			MinimumUpdatePeriod: ptrs.Strptr("PT4S"),
		},
	}
	vodManifest := &Manifest{
		MPD: &mpd.MPD{
			Type: ptrs.Strptr("static"),
		},
	}

	testCases := []struct {
		name     string
		manifest *Manifest
		config   *Config
		cont     bool
		dur      time.Duration
	}{
		{
			name:     "live_default",
			manifest: liveManifest,
			config: &Config{
				DefaultInterval:             5 * time.Second,
				PrioritizeSuggestedInterval: false,
			},
			cont: true,
			dur:  5 * time.Second,
		},
		{
			name:     "live_suggested",
			manifest: liveManifest,
			config: &Config{
				DefaultInterval:             5 * time.Second,
				PrioritizeSuggestedInterval: true,
			},
			cont: true,
			dur:  4 * time.Second,
		},
		{
			name:     "vod_not_terminate",
			manifest: vodManifest,
			config: &Config{
				DefaultInterval:             5 * time.Second,
				PrioritizeSuggestedInterval: true,
			},
			cont: true,
			dur:  5 * time.Second,
		},
		{
			name:     "vod_terminate",
			manifest: vodManifest,
			config: &Config{
				DefaultInterval:             5 * time.Second,
				PrioritizeSuggestedInterval: true,
				TerminateIfVOD:              true,
			},
			cont: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewMonitor(tc.config).(*monitor)
			cont, dur := m.dashWaitDuration(tc.manifest)
			assert.Equal(t, tc.cont, cont)
			assert.Equal(t, tc.dur, dur)
		})
	}
}
