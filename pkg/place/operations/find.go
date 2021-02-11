package operations

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/shareed2k/honey/pkg/place"
	"github.com/shareed2k/honey/pkg/place/cache"
	"github.com/shareed2k/honey/pkg/place/printers"
)

var (
	log = logrus.WithField("operation", "Find")
)

// Find _
func Find(ctx context.Context, backendNames []string, pattern string, force bool, outFormat string, noColor bool) error {
	var backends []place.Backend
	var wg sync.WaitGroup

	cacheDB, err := cache.NewStore()
	if err != nil {
		return err
	}

	defer cacheDB.Close()

	instances := make(place.Printable, 0)
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
				log.Debugf("using cache: %s, pattern %s", name, pattern)

				instances = append(instances, ins...)

				continue
			}
		}

		backends = append(backends, backend)
	}

	for _, b := range backends {
		wg.Add(1)
		go func(ctx context.Context, wg *sync.WaitGroup, backend place.Backend) {
			defer wg.Done()

			log.Debugf("using backend: %s, pattern %s", backend.Name(), pattern)

			ins, err := backend.List(ctx, pattern)
			if err != nil {
				log.Fatal(err)
			}

			// store to cache
			if err := cacheDB.Put(backend.Name(), []byte(backend.CacheKeyName(pattern)), ins); err != nil {
				log.Debugf("can't store cache for (%s) backend", backend.Name())
			}

			instances = append(instances, ins...)
		}(ctx, &wg, b)
	}

	wg.Wait()

	return printers.Print(&printers.PrintInput{
		Data:    instances,
		Format:  outFormat,
		NoColor: noColor,
	})
}
