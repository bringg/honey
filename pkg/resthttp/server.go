package resthttp

import (
	"time"

	"github.com/labstack/echo/v4"

	"github.com/bringg/honey/pkg/place"
	"github.com/bringg/honey/pkg/resthttp/handlers"
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
	e := echo.New()
	e.HideBanner = true

	return &Server{
		echo: e,
	}
}

func (s *Server) Serve() error {
	// Middlewares
	// set copy of config to request context
	s.echo.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx, _ := place.AddConfig(c.Request().Context())
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	})

	// Routes
	s.echo.GET("/backends", handlers.Backends())
	s.echo.GET("/search", handlers.Search())

	return s.echo.Start(":1323")
}
