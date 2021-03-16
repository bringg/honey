package resthttp

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pkg/browser"
	"github.com/sirupsen/logrus"

	"github.com/bringg/honey/pkg/place"
	"github.com/bringg/honey/pkg/resthttp/handlers"
	"github.com/bringg/honey/ui"
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
		echo   *echo.Echo
		Opt    *Options
		useSSL bool
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
		UI                 bool          // enable ui
	}
)

// NewServer _
func NewServer(opt *Options) *Server {
	e := echo.New()
	s := Server{
		Opt:  opt,
		echo: e,
	}

	e.HideBanner = true
	e.Server.MaxHeaderBytes = opt.MaxHeaderBytes
	e.Server.ReadTimeout = opt.ServerReadTimeout
	e.Server.WriteTimeout = opt.ServerWriteTimeout

	s.useSSL = opt.SslKey != ""
	if (opt.SslCert != "") != s.useSSL {
		log.Fatalf("Need both -cert and -key to use SSL")
	}

	if opt.ClientCA != "" {
		if !s.useSSL {
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

	if opt.UI {
		e.Server.BaseContext = func(l net.Listener) context.Context {
			url := s.URL()
			log.Infof("Serving on %s\n", url)

			if err := browser.OpenURL(url); err != nil {
				log.Warn("can't open browser ", err)
			}

			return context.Background()
		}
	}

	return &s
}

func (s *Server) Serve() error {
	basic := s.echo.Group(s.Opt.BaseURL)
	api := basic.Group("/api/v1")

	// Middlewares
	// set copy of config to request context
	api.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx, _ := place.AddConfig(c.Request().Context())
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	})

	if s.Opt.UI {
		// set cors
		basic.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowMethods: []string{"GET"},
			AllowHeaders: []string{
				echo.HeaderAuthorization,
				echo.HeaderOrigin,
				echo.HeaderContentType,
				echo.HeaderAccept,
				echo.HeaderXRequestedWith,
			},
			ExposeHeaders: []string{
				"X-Total-Count",
			},
			MaxAge: 1728000,
		}))

		// set compresses HTTP response using gzip compression scheme
		basic.Use(middleware.GzipWithConfig(middleware.GzipConfig{
			Level: 5,
		}))

		uiHandler := echo.WrapHandler(
			http.StripPrefix(
				s.Opt.BaseURL+"/",
				http.FileServer(http.FS(ui.MustFS())),
			),
		)

		// ui endpoint
		basic.GET("/", uiHandler)
		basic.GET("/*", uiHandler)
	}

	// set basic auth
	if s.Opt.BasicUser != "" {
		basic.Use(middleware.BasicAuthWithConfig(middleware.BasicAuthConfig{
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
	api.GET("/instances", handlers.Instances())

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

// URL returns the serving address of this server
func (s *Server) URL() string {
	proto := "http"
	if s.useSSL {
		proto = "https"
	}

	addr := s.Opt.ListenAddr
	// prefer actual listener address if using ":port" or "addr:0"
	useActualAddress := addr == "" || addr[0] == ':' || addr[len(addr)-1] == ':' || strings.HasSuffix(addr, ":0")
	if s.echo.Listener != nil && useActualAddress {
		// use actual listener address; required if using 0-port
		// (i.e. port assigned by operating system)
		addr = s.echo.ListenerAddr().String()
	}

	return fmt.Sprintf("%s://%s%s/", proto, addr, s.Opt.BaseURL)
}
