package resthttp

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"

	"github.com/bringg/honey/pkg/place"
	"github.com/bringg/honey/pkg/resthttp/handlers"
)

var (
	DefaultOpt = Options{
		ListenAddr:         "localhost:8080",
		Realm:              "honey",
		ServerReadTimeout:  60 * time.Second,
		ServerWriteTimeout: 60 * time.Second,
		MaxHeaderBytes:     4096,
	}

	log = logrus.WithField("where", "server")
)

type (
	Server struct {
		echo *echo.Echo
		Opt  *Options
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
		Realm              string        // realm for authentication
		BasicUser          string        // single username for basic auth
		BasicPass          string        // password for BasicUser
	}
)

// NewServer _
func NewServer(opt *Options) *Server {
	e := echo.New()

	e.HideBanner = true
	e.Server.MaxHeaderBytes = opt.MaxHeaderBytes
	e.Server.ReadTimeout = opt.ServerReadTimeout
	e.Server.WriteTimeout = opt.ServerWriteTimeout

	useSSL := opt.SslKey != ""
	if (opt.SslCert != "") != useSSL {
		log.Fatalf("Need both -cert and -key to use SSL")
	}

	if opt.ClientCA != "" {
		if !useSSL {
			log.Fatalf("Can't use --client-ca without --cert and --key")
		}

		certpool := x509.NewCertPool()
		pem, err := ioutil.ReadFile(opt.ClientCA)
		if err != nil {
			log.Fatalf("Failed to read client certificate authority: %v", err)
		}

		if !certpool.AppendCertsFromPEM(pem) {
			log.Fatalf("Can't parse client certificate authority")
		}

		e.Server.TLSConfig.ClientCAs = certpool
		e.Server.TLSConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	// If a Base URL is set then serve from there
	opt.BaseURL = strings.Trim(opt.BaseURL, "/")
	if opt.BaseURL != "" {
		opt.BaseURL = "/" + opt.BaseURL
	}

	return &Server{
		echo: e,
		Opt:  opt,
	}
}

func (s *Server) Serve() error {
	api := s.echo.Group(s.Opt.BaseURL + "/api/v1")

	// Middlewares
	// set copy of config to request context
	api.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx, _ := place.AddConfig(c.Request().Context())
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	})

	// set compresses HTTP response using gzip compression scheme
	api.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Level: 5,
	}))

	// set basic auth
	if s.Opt.BasicUser != "" {
		api.Use(middleware.BasicAuthWithConfig(middleware.BasicAuthConfig{
			Realm: s.Opt.Realm,
			Validator: func(username, password string, c echo.Context) (bool, error) {
				if strings.Compare(username, s.Opt.BasicUser) == 0 &&
					strings.Compare(username, s.Opt.BasicUser) == 0 {
					return true, nil
				}

				return false, nil
			},
		}))
	}

	// Routes
	api.GET("/backends", handlers.Backends())
	api.GET("/search", handlers.Search())

	// Start server
	go func() {
		if err := s.echo.Start(s.Opt.ListenAddr); err != nil && err != http.ErrServerClosed {
			log.Fatal("shutting down the server")
		}
	}()

	quit := make(chan os.Signal, 1)

	// interrupt signal sent from terminal
	// sigterm signal sent from kubernetes
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	<-quit

	log.Debug("gracefully shutting down the server")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.echo.Shutdown(ctx); err != nil {
		return err
	}

	return nil
}
