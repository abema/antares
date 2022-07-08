package core

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/abema/antares/internal/thread"
	backoff "github.com/cenkalti/backoff/v4"
	"golang.org/x/sync/errgroup"
)

type cache struct {
	data []byte
	del  bool
}

type SegmentStore interface {
	Exists(url string) bool
	Load(url string) ([]byte, bool)
}

type mutableSegmentStore interface {
	SegmentStore
	Sync(ctx context.Context, urls []string) error
}

type segmentStore struct {
	httpClient client
	backoff    backoff.BackOff
	cacheMap   map[string]*cache
	timeout    time.Duration
	maxConc    int
}

func newSegmentStore(
	httpClient client,
	timeout time.Duration,
	backoff backoff.BackOff,
	maxConcurrency int,
) mutableSegmentStore {
	return &segmentStore{
		httpClient: httpClient,
		backoff:    backoff,
		cacheMap:   make(map[string]*cache),
		timeout:    timeout,
		maxConc:    maxConcurrency,
	}
}

func (s *segmentStore) Exists(url string) bool {
	_, ok := s.cacheMap[url]
	return ok
}

func (s *segmentStore) Load(url string) ([]byte, bool) {
	seg, ok := s.cacheMap[url]
	return seg.data, ok
}

func (s *segmentStore) Sync(ctx context.Context, urls []string) error {
	for url := range s.cacheMap {
		s.cacheMap[url].del = true
	}

	type result struct {
		url  string
		data []byte
	}
	results := make(chan result, len(urls))
	eg := new(errgroup.Group)
	maxConc := s.maxConc
	if maxConc == 0 {
		maxConc = 1
	}
	limiter := make(chan struct{}, maxConc)
	for i := range urls {
		url := urls[i]
		if seg, ok := s.cacheMap[url]; ok {
			seg.del = false
			continue
		}
		limiter <- struct{}{}
		eg.Go(thread.NoPanic(func() error {
			defer func() {
				<-limiter
			}()
			return backoff.RetryNotify(func() error {
				ctx, cancel := context.WithTimeout(ctx, s.timeout)
				defer cancel()
				data, _, err := s.httpClient.Get(ctx, url)
				if err != nil {
					err = fmt.Errorf("failed to download segment: %s: %w", url, err)
					if ctx.Err() != nil || errors.As(err, &permanentError{}) {
						return backoff.Permanent(err)
					}
					return err
				}
				results <- result{url: url, data: data}
				return nil
			}, s.backoff, func(err error, _ time.Duration) {
				log.Printf("WARN: failed to download segment: %s: %s", url, err)
			})
		}))
	}
	if err := eg.Wait(); err != nil {
		return err
	}
	close(results)
	for res := range results {
		s.cacheMap[res.url] = &cache{data: res.data}
	}
	for url := range s.cacheMap {
		if s.cacheMap[url].del {
			delete(s.cacheMap, url)
		}
	}
	return nil
}
