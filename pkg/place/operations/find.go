package operations

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/bringg/honey/pkg/place"
	"github.com/bringg/honey/pkg/place/cache"
	"github.com/bringg/honey/pkg/place/printers"
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
func Find(ctx context.Context, backendNames []string, pattern string, force bool, outFormat string, noColor bool) error {
	var backends []place.Backend

	cacheDB, err := cache.NewStore()
	if err != nil {
		return err
	}

	defer cacheDB.Close()

	instances := new(ConcurrentSlice)

	for _, name := range backendNames {
		info, err := place.Find(name)
		if err != nil {
			return err
		}

		backend, err := info.NewBackend(ctx, place.ConfigMap(info, info.Name))
		if err != nil {
			return err
		}

		// try to take from cache
		if !force {
			ins := make(place.Printable, 0)
			if err := cacheDB.Get(name, []byte(backend.CacheKeyName(pattern)), &ins); err == nil {
				log.Debugf("using cache: %s, pattern `%s`, found: %d items", name, pattern, len(ins))

				instances.Append(ins)

				continue
			}

			if err != nil {
				log.Debug(err)
			}
		}

		backends = append(backends, backend)
	}

	g, fCtx := errgroup.WithContext(ctx)

	for _, b := range backends {
		g.Go(func(backend place.Backend) func() error {
			return func() error {
				ins, err := backend.List(fCtx, pattern)
				if err != nil {
					return err
				}

				log.Debugf("using backend: %s, pattern `%s`, found: %d items", backend.Name(), pattern, len(ins))

				// store to cache
				if err := cacheDB.Put(backend.Name(), []byte(backend.CacheKeyName(pattern)), ins); err != nil {
					log.Debugf("can't store cache for (%s) backend: %v", backend.Name(), err)
				}

				instances.Append(ins)

				return nil
			}
		}(b))
	}

	if err := g.Wait(); err != nil {
		return err
	}

	return printers.Print(&printers.PrintInput{
		Data:    instances.Items,
		Format:  outFormat,
		NoColor: noColor,
	})
}
