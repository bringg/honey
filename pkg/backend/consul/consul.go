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
		opt Options
		c   *api.Client
	}

	// Options defines the configuration for this backend
	Options struct {
	}
)

// Register with Backend
func init() {
	place.Register(&place.RegInfo{
		Name:        Name,
		Description: "Consul by HashiCorp",
		NewBackend:  NewBackend,
		Options:     []place.Option{},
	})
}

func NewBackend(ctx context.Context, m configmap.Mapper) (place.Backend, error) {
	// Parse config into Options struct
	opt := new(Options)
	err := configstruct.Set(m, opt)
	if err != nil {
		return nil, err
	}

	// Get a new client
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return nil, err
	}

	return &Backend{
		opt: *opt,
		c:   client,
	}, nil
}

func (b *Backend) Name() string {
	return Name
}

func (b *Backend) CacheKeyName(pattern string) string {
	return fmt.Sprintf("%s", pattern)
}

func (b *Backend) List(ctx context.Context, pattern string) (place.Printable, error) {
	catalog := b.c.Catalog()

	nodes, _, err := catalog.Nodes(&api.QueryOptions{
		Filter: fmt.Sprintf(`Node contains "%s"`, pattern),
	})
	if err != nil {
		return nil, err
	}

	instances := make([]*place.Instance, len(nodes))
	for i, node := range nodes {
		cn, _, err := catalog.NodeServiceList(node.Node, &api.QueryOptions{})
		if err != nil {
			return nil, err
		}

		publicIP := ""
		if wan, ok := cn.Node.TaggedAddresses["wan"]; ok {
			publicIP = wan
		}

		instances[i] = &place.Instance{
			Model: place.Model{
				BackendName: Name,
				ID:          cn.Node.ID,
				Name:        cn.Node.Node,
				Type:        "node",
				Status:      "",
				PrivateIP:   cn.Node.Address,
				PublicIP:    publicIP,
			},
			Raw: cn,
		}
	}

	return instances, nil
}
