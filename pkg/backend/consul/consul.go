package consul

import (
	"context"
	"fmt"

	"github.com/hashicorp/consul/api"
	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/rclone/rclone/fs/config/configstruct"
	"github.com/sirupsen/logrus"

	"github.com/bringg/honey/pkg/place"
)

const Name = "consul"

var (
	log = logrus.WithField("backend", Name)
)

type (
	Backend struct {
		opt    Options
		client *api.Client
	}

	// Options defines the configuration for this backend
	Options struct {
		Address            string `config:"address"`
		Schema             string `config:"scheme"`
		Datacenter         string `config:"datacenter"`
		TokenFile          string `config:"token_file"`
		Token              string `config:"token"`
		Namespace          string `config:"namespace"`
		AuthBasicUsername  string `config:"auth_basic_username"`
		AuthBasicPassword  string `config:"auth_basic_password"`
		HTTPTLSServerName  string `config:"http_tls_server_name"`
		HTTPCAFile         string `config:"http_ca_file"`
		HTTPCAPath         string `config:"http_ca_path"`
		HTTPClientCert     string `config:"http_client_cert"`
		HTTPClientKey      string `config:"http_client_key"`
		HTTPSSLEnvName     bool   `config:"http_ssl_env_name"`
		InsecureSkipVerify bool   `config:"insecure_skip_verify"`
	}
)

// Register with Backend
func init() {
	place.Register(&place.RegInfo{
		Name:        Name,
		Description: "Consul by HashiCorp",
		NewBackend:  NewBackend,
		Options: []place.Option{
			{
				Name: "address",
				Help: "Address of the Consul server",
			},
			{
				Name: "scheme",
				Help: "URI scheme for the Consul server",
			},
			{
				Name: "datacenter",
				Help: "Datacenter to use",
			},
			{
				Name: "token_file",
				Help: "TokenFile is a file containing the current token to use for this client",
			},
			{
				Name: "token",
				Help: "Token is used to provide a per-request ACL token",
			},
			{
				Name: "namespace",
				Help: "Namespace is the name of the namespace to send along for the request when no other Namespace is present in the QueryOptions",
			},
			{
				Name: "auth_basic_username",
				Help: "Username is the auth info to use for http access",
			},
			{
				Name: "auth_basic_password",
				Help: "Password is the auth info to use for http access",
			},
			{
				Name:    "http_ssl_env_name",
				Help:    "Http_ssl_env_name sets whether or not to use HTTPS",
				Default: false,
			},
			{
				Name: "http_tls_server_name",
				Help: "Address is the optional address of the Consul server. The port, if any will be removed from here and this will be set to the ServerName of the resulting config",
			},
			{
				Name: "http_ca_file",
				Help: "CAFile is the optional path to the CA certificate used for Consul communication, defaults to the system bundle if not specified",
			},
			{
				Name: "http_ca_path",
				Help: "HTTPCAPath defines an environment variable name which sets the path to a directory of CA certs to use for talking to Consul over TLS",
			},
			{
				Name: "http_client_cert",
				Help: "CertFile is the optional path to the certificate for Consul communication. If this is set then you need to also set KeyFile",
			},
			{
				Name: "http_client_key",
				Help: "KeyFile is the optional path to the private key for Consul communication. If this is set then you need to also set CertFile",
			},
			{
				Name:    "insecure_skip_verify",
				Help:    "InsecureSkipVerify if set to true will disable TLS host verification",
				Default: false,
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

	cfg := api.DefaultConfig()

	if opt.Address != "" {
		cfg.Address = opt.Address
	}

	if opt.HTTPSSLEnvName {
		cfg.Scheme = "https"
	}

	if opt.HTTPTLSServerName != "" {
		cfg.TLSConfig.Address = opt.HTTPTLSServerName
	}

	if opt.HTTPCAFile != "" {
		cfg.TLSConfig.CAFile = opt.HTTPCAFile
	}

	if opt.HTTPCAPath != "" {
		cfg.TLSConfig.CAPath = opt.HTTPCAPath
	}

	if opt.HTTPClientCert != "" {
		cfg.TLSConfig.CertFile = opt.HTTPClientCert
	}

	if opt.HTTPClientKey != "" {
		cfg.TLSConfig.KeyFile = opt.HTTPClientKey
	}

	if !opt.InsecureSkipVerify {
		cfg.TLSConfig.InsecureSkipVerify = true
	}

	if opt.AuthBasicUsername != "" || opt.AuthBasicPassword != "" {
		cfg.HttpAuth = &api.HttpBasicAuth{
			Username: opt.AuthBasicUsername,
			Password: opt.AuthBasicPassword,
		}
	}

	// Get a new client
	client, err := api.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	return &Backend{
		opt:    *opt,
		client: client,
	}, nil
}

func (b *Backend) Name() string {
	return Name
}

func (b *Backend) CacheKeyName(pattern string) string {
	return fmt.Sprintf("%s", pattern)
}

func (b *Backend) List(ctx context.Context, pattern string) (place.Printable, error) {
	nodes, _, err := b.client.Catalog().Nodes(&api.QueryOptions{
		Filter: fmt.Sprintf(`Node contains "%s"`, pattern),
	})
	if err != nil {
		return nil, err
	}

	instances := make([]*place.Instance, len(nodes))
	for i, node := range nodes {
		hc, _, err := b.client.Health().Node(node.Node, &api.QueryOptions{})
		if err != nil {
			return nil, err
		}

		privateIP := node.Address
		publicIP := ""
		if wan, ok := node.TaggedAddresses["wan"]; ok && privateIP != wan {
			publicIP = wan
		}

		instances[i] = &place.Instance{
			Model: place.Model{
				BackendName: Name,
				ID:          node.ID,
				Name:        node.Node,
				Type:        "node",
				Status:      hc.AggregatedStatus(),
				PrivateIP:   privateIP,
				PublicIP:    publicIP,
			},
			Raw: node,
		}
	}

	return instances, nil
}
