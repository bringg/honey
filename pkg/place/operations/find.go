package operations

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/bringg/honey/pkg/place"
	"github.com/bringg/honey/pkg/place/cache"
)

var (
	log = logrus.WithField("operation", "Find")
)

type (
	ConcurrentSlice struct {
		sync.RWMutex
		Items place.Printable
	}
)

func (cs *ConcurrentSlice) Append(item place.Printable) {
	cs.Lock()
	defer cs.Unlock()

	cs.Items = append(cs.Items, item...)
}

// Find _
func Find(ctx context.Context, backendNames []string, pattern string) (place.Printable, error) {
	backends := make(map[string]place.Backend)

	cacheDB, err := cache.NewStore()
	if err != nil {
		return nil, err
	}

	defer cacheDB.Close()

	instances := new(ConcurrentSlice)
	ci := place.GetConfig(ctx)

	for _, name := range backendNames {
		bucketName := name

		m := place.ConfigMap(nil, name)
		if bName, ok := m.Get("type"); ok {
			name = bName
		}

		info, err := place.Find(name)
		if err != nil {
			return nil, err
		}

		backend, err := info.NewBackend(ctx, m)
		if err != nil {
			return nil, errors.Wrap(err, name)
		}

		// try to take from cache
		if !ci.NoCache {
			ins := make(place.Printable, 0)
			if err := cacheDB.Get(bucketName, []byte(backend.CacheKeyName(pattern)), &ins); err == nil {
				log.Debugf("using cache: %s, provider %s, pattern `%s`, found: %d items", bucketName, name, pattern, len(ins))

				instances.Append(ins)

				continue
			}

			if err != nil {
				log.Debug(err)
			}
		}

		backends[bucketName] = backend
	}

	g, fCtx := errgroup.WithContext(ctx)

	for bucketName, b := range backends {
		g.Go(func(bucketName string, backend place.Backend) func() error {
			return func() error {
				ins, err := backend.List(fCtx, pattern)
				if err != nil {
					return errors.Wrap(err, b.Name())
				}

				log.Debugf("using backend: %s, provider %s, pattern `%s`, found: %d items", bucketName, backend.Name(), pattern, len(ins))

				// store to cache
				if err := cacheDB.Put(bucketName, []byte(backend.CacheKeyName(pattern)), ins); err != nil {
					log.Debugf("can't store cache for (%s) backend: %v", bucketName, err)
				}

				instances.Append(ins)

				return nil
			}
		}(bucketName, b))
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return instances.Items, nil
}
