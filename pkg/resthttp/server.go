package resthttp

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/bringg/honey/pkg/place/operations"
)

type (
	Server struct {
		echo *echo.Echo
	}

	Options struct {
		ListenAddr         string        // Port to listen on
		BaseURL            string        // prefix to strip from URLs
		ServerReadTimeout  time.Duration // Timeout for server reading data
		ServerWriteTimeout time.Duration // Timeout for server writing data
		MaxHeaderBytes     int           // Maximum size of request header
		SslCert            string        // SSL PEM key (concatenation of certificate and CA certificate)
		SslKey             string        // SSL PEM Private key
		ClientCA           string        // Client certificate authority to verify clients with
		HtPasswd           string        // htpasswd file - if not provided no authentication is done
		Realm              string        // realm for authentication
		BasicUser          string        // single username for basic auth if not using Htpasswd
		BasicPass          string        // password for BasicUser
	}
)

func NewServer(opts *Options) *Server {
	return &Server{
		echo: echo.New(),
	}
}

func (s *Server) Serve() error {
	// Routes
	s.echo.GET("/backends", func(c echo.Context) error {
		filter := c.QueryParam("filter")
		backends := c.Request().URL.Query()["backend"]

		instances, err := operations.Find(c.Request().Context(), backends, filter)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		flattenData, err := instances.FlattenData()
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		cleanedData, err := flattenData.Filter(instances.Headers())
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, cleanedData)
	})

	return s.echo.Start(":1323")
}
