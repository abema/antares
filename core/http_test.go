package core

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPermanentError(t *testing.T) {
	parent := errors.New("foo")
	assert.False(t, errors.As(parent, &permanentError{}))
	err := newPermanentError(parent)
	assert.True(t, errors.As(err, &permanentError{}))
	assert.True(t, errors.Is(err, parent))
	assert.False(t, errors.Is(err, errors.New("bar")))
}

func TestClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "antares test", r.Header.Get("User-Agent"))
		switch r.URL.Path {
		case "/redirect":
			http.Redirect(w, r, "/hello", http.StatusFound)
		case "/hello":
			w.Write([]byte("hello antares"))
		case "/not_found":
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("not found"))
		}
	}))
	header := http.Header{
		"User-Agent": []string{"antares test"},
	}

	t.Run("200_ok", func(t *testing.T) {
		var handle bool
		client := newClient(&http.Client{}, header, func(file *File) {
			require.Equal(t, 200, file.StatusCode)
			require.Equal(t, []byte("hello antares"), file.Body)
			handle = true
		})
		data, loc, err := client.Get(context.Background(), server.URL+"/hello")
		require.NoError(t, err)
		assert.Equal(t, server.URL+"/hello", loc)
		assert.Equal(t, "hello antares", string(data))
		assert.True(t, handle)
	})

	t.Run("302_found", func(t *testing.T) {
		var handle bool
		client := newClient(&http.Client{}, header, func(file *File) {
			require.Equal(t, 200, file.StatusCode)
			require.Equal(t, []byte("hello antares"), file.Body)
			handle = true
		})
		data, loc, err := client.Get(context.Background(), server.URL+"/redirect")
		require.NoError(t, err)
		assert.Equal(t, server.URL+"/hello", loc)
		assert.Equal(t, "hello antares", string(data))
		assert.True(t, handle)
	})

	t.Run("404_not_found", func(t *testing.T) {
		var handle bool
		client := newClient(&http.Client{}, header, func(file *File) {
			require.Equal(t, 404, file.StatusCode)
			require.Equal(t, []byte("not found"), file.Body)
			handle = true
		})
		_, _, err := client.Get(context.Background(), server.URL+"/not_found")
		require.Error(t, err)
		assert.True(t, handle)
	})
}

func TestRedirectKeeper(t *testing.T) {
	accessLog := make([]string, 0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "antares test", r.Header.Get("User-Agent"))
		switch r.URL.Path {
		case "/redirect":
			http.Redirect(w, r, "/hello", http.StatusFound)
		case "/hello":
			w.Write([]byte("hello antares"))
		}
		accessLog = append(accessLog, r.URL.Path)
	}))
	client := newRedirectKeeper(newClient(&http.Client{}, http.Header{
		"User-Agent": []string{"antares test"},
	}, nil))
	data, loc, err := client.Get(context.Background(), server.URL+"/redirect")
	require.NoError(t, err)
	assert.Equal(t, server.URL+"/hello", loc)
	assert.Equal(t, "hello antares", string(data))
	data, loc, err = client.Get(context.Background(), server.URL+"/redirect")
	require.NoError(t, err)
	assert.Equal(t, server.URL+"/hello", loc)
	assert.Equal(t, "hello antares", string(data))
	assert.Equal(t, []string{"/redirect", "/hello", "/hello"}, accessLog)
}
