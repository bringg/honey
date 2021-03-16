package handlers

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"

	"github.com/bringg/honey/pkg/config"
	"github.com/bringg/honey/pkg/place"
)

func Backends() echo.HandlerFunc {
	customBackends := make(map[string]Backend, 0)
	if err := config.BackendListUnmarshal(&customBackends); err != nil {
		logrus.Fatal(err)
	}

	backends := make([]Backend, 0)
	i := 1
	for name, backend := range customBackends {
		if backend.Type == "" {
			continue
		}

		backend.ID = i
		backend.Name = name

		backends = append(backends, backend)

		i++
	}

	for _, name := range place.BackendNames() {
		backends = append(backends, Backend{
			ID:   i,
			Name: name,
			Type: name,
		})

		i++
	}

	count := strconv.Itoa(i - 1)
	return func(c echo.Context) error {
		// set custom count header for ui
		c.Response().Header().Add("X-Total-Count", count)

		return c.JSONPretty(http.StatusOK, backends, "   ")
	}
}
