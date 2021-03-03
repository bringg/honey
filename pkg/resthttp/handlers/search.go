package handlers

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/bringg/honey/pkg/place/operations"
)

func Search() echo.HandlerFunc {
	return func(c echo.Context) error {
		filter := c.QueryParam("filter")
		backends := c.Request().URL.Query()["backend"]
		keys := c.Request().URL.Query()["key"]

		instances, err := operations.Find(c.Request().Context(), backends, filter)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		flattenData, err := instances.FlattenData()
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		if len(keys) == 0 {
			keys = instances.Headers()
		}

		cleanedData, err := flattenData.Filter(keys)
		if err != nil {
			return err
		}

		c.Response().Header().Add("X-Total-Count", strconv.Itoa(len(cleanedData)))

		return c.JSONPretty(http.StatusOK, cleanedData, "   ")
	}
}
