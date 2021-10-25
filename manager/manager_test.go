package manager

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"github.com/abema/antares/core"
	"github.com/stretchr/testify/require"
)

func TestManager(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("#EXTM3U\n"))
	}))
	m := NewManager(&Config{})
	m.Add("a", core.NewConfig(server.URL, core.StreamTypeHLS))
	require.NotNil(t, m.Get("a"))
	require.Nil(t, m.Get("b"))
	m.Add("b", core.NewConfig(server.URL, core.StreamTypeHLS))
	require.NotNil(t, m.Get("a"))
	require.NotNil(t, m.Get("b"))
	m.Get("a").Terminate()
	require.NotNil(t, m.Get("a"))
	added, removed := m.Batch(map[string]*core.Config{
		"b": core.NewConfig(server.URL, core.StreamTypeHLS),
		"c": core.NewConfig(server.URL, core.StreamTypeHLS),
		"d": core.NewConfig(server.URL, core.StreamTypeHLS),
	})
	sort.Strings(added)
	sort.Strings(removed)
	require.Equal(t, []string{"c", "d"}, added)
	require.Equal(t, []string{"a"}, removed)
	require.Nil(t, m.Get("a"))
	require.NotNil(t, m.Get("b"))
	require.NotNil(t, m.Get("c"))
	require.NotNil(t, m.Get("d"))
	require.Len(t, m.Map(), 3)
	m.Remove("b")
	require.Nil(t, m.Get("b"))
	removed = m.RemoveAll()
	sort.Strings(removed)
	require.Equal(t, []string{"c", "d"}, removed)
	require.Empty(t, m.Map())
}

func TestManagerWithAutoRemove(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("#EXTM3U\n"))
	}))
	m := NewManager(&Config{AutoRemove: true})
	m.Add("a", core.NewConfig(server.URL, core.StreamTypeHLS))
	require.NotNil(t, m.Get("a"))
	m.Get("a").Terminate()
	time.Sleep(10 * time.Millisecond)
	require.Nil(t, m.Get("a"))
}
