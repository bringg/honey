package macstadium

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/rclone/rclone/fs/config/configstruct"
	"github.com/rclone/rclone/fs/config/obscure"
	"github.com/rclone/rclone/fs/fserrors"
	"github.com/rclone/rclone/fs/fshttp"
	"github.com/rclone/rclone/lib/pacer"
	"github.com/rclone/rclone/lib/rest"
	"github.com/sirupsen/logrus"

	"github.com/bringg/honey/pkg/place"
)

const (
	Name             = "macstadium"
	defaultEndpoint  = "https://api.macstadium.com"
	minSleep         = 10 * time.Millisecond
	maxSleep         = 5 * time.Minute
	decayConstant    = 1 // bigger for slower decay, exponential
	retryAfterHeader = "Retry-After"
)

var (
	log = logrus.WithField("backend", Name)
	// retryErrorCodes is a slice of error codes that we will retry
	retryErrorCodes = []int{
		// 401, // Unauthorized (e.g. "Token has expired")
		408, // Request Timeout
		429, // Rate exceeded.
		500, // Get occasional 500 Internal Server Error
		503, // Service Unavailable
		504, // Gateway Time-out
	}
)

type (
	Backend struct {
		opt    Options
		client *rest.Client
		pacer  *fs.Pacer // To pace and retry the API calls
	}

	// Options defines the configuration for this backend
	Options struct {
		Endpoint string `config:"endpoint"`
		UserName string `config:"username"`
		Password string `config:"password"`
	}

	Server struct {
		ID     string        `json:"id"`
		Name   string        `json:"name"`
		IP     string        `json:"ip"`
		Status *ServerStatus `json:"status"`
	}

	ServerStatus struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Power string `json:"power"`
	}
)

// Register with Backend
func init() {
	place.Register(&place.RegInfo{
		Name:        Name,
		Description: "MacStadium Mac Servers",
		NewBackend:  NewBackend,
		Options: []place.Option{
			{
				Name:     "username",
				Help:     "Username Basic Authentication",
				Required: true,
			}, {
				Name:       "password",
				Help:       "Password Basic Authentication \nInput to this must be obscured\n\necho \"secretpassword\" | honey obscure -",
				Required:   true,
				IsPassword: true,
			},
			{
				Name:    "endpoint",
				Help:    "Endpoint for the service",
				Default: defaultEndpoint,
			},
		},
	})
}

func NewBackend(ctx context.Context, m configmap.Mapper) (place.Backend, error) {
	// Parse config into Options struct
	opt := new(Options)
	err := configstruct.Set(m, opt)
	if err != nil {
		return nil, err
	}

	if opt.UserName == "" {
		return nil, errors.New("username not found")
	}

	if opt.Password == "" {
		return nil, errors.New("password not found")
	}

	password, err := obscure.Reveal(opt.Password)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decrypt password")
	}

	opt.Password = password

	if opt.Endpoint == "" {
		opt.Endpoint = defaultEndpoint
	}

	return &Backend{
		opt:    *opt,
		pacer:  fs.NewPacer(ctx, pacer.NewDefault(pacer.MinSleep(minSleep), pacer.MaxSleep(maxSleep), pacer.DecayConstant(decayConstant))),
		client: rest.NewClient(fshttp.NewClient(ctx)),
	}, nil
}

func (b *Backend) Name() string {
	return Name
}

func (b *Backend) CacheKeyName(pattern string) string {
	return pattern
}

func (b *Backend) List(ctx context.Context, backendName string, pattern string) (place.Printable, error) {
	servers, err := b.listAllServers(ctx)
	if err != nil {
		return nil, err
	}

	filter, err := regexp.Compile(fmt.Sprintf("(?i).*%s.*", pattern))
	if err != nil {
		return nil, errors.Wrap(err, "failed to compile regular expression from query")
	}

	instances := make([]*place.Instance, 0)
	for _, server := range servers {
		status, err := b.serverStatus(ctx, server.ID)
		if err != nil {
			return nil, err
		}

		server.Status = status

		if !filter.MatchString(server.Name) {
			continue
		}

		instances = append(instances, &place.Instance{
			Model: place.Model{
				BackendName: backendName,
				ID:          server.ID,
				Name:        server.Name,
				Type:        "macOs",
				Status:      status.Power,
				PrivateIP:   "",
				PublicIP:    server.IP,
			},
			Raw: server,
		})
	}

	return instances, nil
}

func (b *Backend) listAllServers(ctx context.Context) ([]*Server, error) {
	opts := rest.Opts{
		Method:   http.MethodGet,
		Path:     "/core/api/servers",
		RootURL:  b.opt.Endpoint,
		UserName: b.opt.UserName,
		Password: b.opt.Password,
	}

	servers := make([]*Server, 0)
	if err := b.pacer.Call(func() (bool, error) {
		resp, err := b.client.CallJSON(ctx, &opts, nil, &servers)
		return b.shouldRetry(resp, err)
	}); err != nil {
		return nil, errors.Wrap(err, "failed to get server list")
	}

	return servers, nil
}

func (b *Backend) serverStatus(ctx context.Context, id string) (*ServerStatus, error) {
	opts := rest.Opts{
		Method:   http.MethodGet,
		Path:     "/core/api/servers/" + id,
		RootURL:  b.opt.Endpoint,
		UserName: b.opt.UserName,
		Password: b.opt.Password,
	}

	status := &ServerStatus{}
	if err := b.pacer.Call(func() (bool, error) {
		resp, err := b.client.CallJSON(ctx, &opts, nil, status)
		return b.shouldRetry(resp, err)
	}); err != nil {
		return nil, errors.Wrap(err, "failed to get server status")
	}

	return status, nil
}

// shouldRetry returns a boolean as to whether this resp and err
// deserve to be retried.  It returns the err as a convenience
func (b *Backend) shouldRetry(resp *http.Response, err error) (bool, error) {
	if resp != nil && resp.StatusCode == 401 {
		log.Debugf("Unauthorized: %v", err)

		return false, err
	}
	// For 429 or 503 errors look at the Retry-After: header and
	// set the retry appropriately, starting with a minimum of 1
	// second if it isn't set.
	if resp != nil && (resp.StatusCode == 429 || resp.StatusCode == 503) {
		var retryAfter = 1
		retryAfterString := resp.Header.Get(retryAfterHeader)
		if retryAfterString != "" {
			var err error
			retryAfter, err = strconv.Atoi(retryAfterString)
			if err != nil {
				log.Errorf("Malformed %s header %q: %v", retryAfterHeader, retryAfterString, err)
			}
		}
		return true, pacer.RetryAfterError(err, time.Duration(retryAfter)*time.Second)
	}
	return fserrors.ShouldRetry(err) || fserrors.ShouldRetryHTTP(resp, retryErrorCodes), err
}
