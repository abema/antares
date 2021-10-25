package core

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

const maxRedirectCount = 10

type permanentError struct {
	parent error
}

func newPermanentError(parent error) error {
	return permanentError{parent: parent}
}

func (err permanentError) Error() string {
	return err.parent.Error()
}

func (err permanentError) Unwrap() error {
	return err.parent
}

type client interface {
	Get(ctx context.Context, url string) ([]byte, string, error)
}

type simpleClient struct {
	bare    *http.Client
	header  http.Header
	handler func(file *File)
}

func newClient(bareClient *http.Client, header http.Header, handler func(file *File)) client {
	return &simpleClient{
		bare:    bareClient,
		header:  header,
		handler: handler,
	}
}

func (c *simpleClient) Get(ctx context.Context, url string) ([]byte, string, error) {
	via := url
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, "", newPermanentError(err)
	}
	req.Header = c.header

	bare := *c.bare
	bare.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		url = req.URL.String()
		if c.bare.CheckRedirect != nil {
			return c.bare.CheckRedirect(req, via)
		}
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		return nil
	}

	requestTime := time.Now()
	resp, err := bare.Do(req)
	if err != nil {
		return nil, "", err
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	elapsed := time.Now().Sub(requestTime)
	if c.handler != nil {
		c.handler(&File{
			Meta: Meta{
				URL:              url,
				Via:              via,
				RequestHeader:    req.Header,
				ResponseHeader:   resp.Header,
				Status:           resp.Status,
				StatusCode:       resp.StatusCode,
				Proto:            resp.Proto,
				ProtoMajor:       resp.ProtoMajor,
				ProtoMinor:       resp.ProtoMinor,
				ContentLength:    resp.ContentLength,
				TransferEncoding: resp.TransferEncoding,
				Uncompressed:     resp.Uncompressed,
				RequestTimestamp: requestTime,
				DownloadTime:     elapsed,
			},
			Body: data,
		})
	}
	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			err = newPermanentError(err)
		}
		return nil, "", err
	}
	return data, url, nil
}

type redirectKeeper struct {
	client
	redirectMap map[string]string
	mutex       sync.RWMutex
}

func newRedirectKeeper(client client) *redirectKeeper {
	return &redirectKeeper{
		client:      client,
		redirectMap: make(map[string]string),
	}
}

func (c *redirectKeeper) Get(ctx context.Context, url string) ([]byte, string, error) {
	via := url
	if loc := c.loadCache(url); loc != "" {
		url = loc
	}
	data, loc, err := c.client.Get(ctx, url)
	if via != loc {
		c.storeCache(via, loc)
	}
	return data, loc, err
}

func (c *redirectKeeper) loadCache(url string) string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.redirectMap[url]
}

func (c *redirectKeeper) storeCache(via, url string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.redirectMap[via] = url
}
