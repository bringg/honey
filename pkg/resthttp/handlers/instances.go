package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	lru "github.com/hnlq715/golang-lru"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"github.com/vcraescu/go-paginator/v2/adapter"

	"github.com/bringg/honey/pkg/place"
	"github.com/bringg/honey/pkg/place/operations"
)

const (
	lrySize      = 100
	defaultLimit = 10
)

var (
	log      = logrus.WithField("server", "Search")
	lruCache = mustCreateLRU()
)

func Instances() echo.HandlerFunc {
	return func(c echo.Context) error {
		instances, err := getInstances(c)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		return c.JSONPretty(http.StatusOK, instances, "   ")
	}
}

func getPositiveInt(i string) int {
	o, _ := strconv.Atoi(i)
	if o < 0 {
		return 0
	}

	return o
}

func mustCreateLRU() *lru.ARCCache {
	l, err := lru.NewARCWithExpire(
		lrySize,
		place.GetConfig(context.Background()).CacheTTL,
	)
	if err != nil {
		log.Fatal(err)
	}

	return l
}

func lruKey(c echo.Context) string {
	return fmt.Sprintf("%s:%s", c.QueryParam("filter"), strings.Join(c.Request().URL.Query()["backend"], ":"))
}

func getInstances(c echo.Context) ([]map[string]interface{}, error) {
	filter := c.QueryParam("filter")
	backends := c.Request().URL.Query()["backend"]
	keys := c.Request().URL.Query()["key"]
	key := lruKey(c)

	var cleanedData []map[string]interface{}
	if items, ok := lruCache.Get(key); ok {
		cleanedData = items.([]map[string]interface{})
	} else {
		instances, err := operations.Find(c.Request().Context(), backends, filter)
		if err != nil {
			return nil, err
		}

		flattenData, err := instances.FlattenData()
		if err != nil {
			return nil, err
		}

		if len(keys) == 0 {
			keys = instances.Headers()
		}

		cleanedData, err = flattenData.Filter(append(keys, "raw"))
		if err != nil {
			return nil, err
		}

		lruCache.Add(key, cleanedData)
	}

	var data []map[string]interface{}
	adp := adapter.NewSliceAdapter(cleanedData)

	limit := getPositiveInt(c.QueryParam("_end"))
	if limit == 0 {
		limit = defaultLimit
	}
	offset := getPositiveInt(c.QueryParam("_start"))
	if err := adp.Slice(offset, limit-offset, &data); err != nil {
		return nil, err
	}

	total, err := adp.Nums()
	if err != nil {
		return nil, err
	}

	c.Response().Header().Add("X-Total-Count", strconv.FormatInt(total, 10))

	return data, nil
}
