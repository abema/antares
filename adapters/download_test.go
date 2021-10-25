package adapters

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/abema/antares/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOnDownloadPathSuffixFilter(t *testing.T) {
	testCases := []struct {
		url      string
		expected int
	}{
		{url: "http://localhost/foo.ts", expected: 1},
		{url: "http://localhost/foo.mp4", expected: 1},
		{url: "http://localhost/foo.m3u8", expected: 0},
		{url: "http://localhost/foo.mpd", expected: 0},
	}
	for _, tc := range testCases {
		t.Run(tc.url, func(t *testing.T) {
			var called int
			OnDownloadPathSuffixFilter(func(file *core.File) {
				called++
			}, ".ts", ".mp4")(&core.File{
				Meta: core.Meta{URL: tc.url},
			})
			assert.Equal(t, tc.expected, called)
		})
	}
}

func TestLocalFileExporter(t *testing.T) {
	dir, err := ioutil.TempDir("", "antares-test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	file := &core.File{
		Meta: core.Meta{
			URL:        "https://foo/bar.mp4",
			Status:     "200 OK",
			StatusCode: 200,
			Proto:      "HTTP/1.0",
		},
		Body: []byte("foo"),
	}
	LocalFileExporter(dir, false)(file)
	f, err := os.Open(path.Join(dir, "foo/bar.mp4"))
	require.NoError(t, err)
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, file.Body, b)
	_, err = os.Open(path.Join(dir, "foo/bar.mp4-meta.json"))
	require.Error(t, err)
}

func TestLocalFileExporterWithMeta(t *testing.T) {
	dir, err := ioutil.TempDir("", "antares-test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	file := &core.File{
		Meta: core.Meta{
			URL:        "https://foo/bar.mp4",
			Status:     "200 OK",
			StatusCode: 200,
			Proto:      "HTTP/1.0",
		},
		Body: []byte("foo"),
	}
	LocalFileExporter(dir, true)(file)
	f, err := os.Open(path.Join(dir, "foo/bar.mp4"))
	require.NoError(t, err)
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, file.Body, b)
	m, err := os.Open(path.Join(dir, "foo/bar.mp4-meta.json"))
	require.NoError(t, err)
	defer m.Close()
	var meta core.Meta
	require.NoError(t, json.NewDecoder(m).Decode(&meta))
	assert.Equal(t, file.Meta, meta)
}
