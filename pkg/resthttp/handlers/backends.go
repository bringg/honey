package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"

	"github.com/bringg/honey/pkg/config"
	"github.com/bringg/honey/pkg/place"
)

func Backends() echo.HandlerFunc {
	res := make(map[string]CustomBackend, 0)
	if err := config.BackendListUnmarshal(&res); err != nil {
		logrus.Fatal(err)
	}

	customBackends := make([]CustomBackend, 0)
	for name, backend := range res {
		backend.Name = name
		customBackends = append(customBackends, backend)
	}

	backends := &BackendsResponse{
		CustomBackends: customBackends,
		Backends:       place.BackendNames(),
	}

	return func(c echo.Context) error {
		return c.JSONPretty(http.StatusOK, backends, "   ")
	}
}
