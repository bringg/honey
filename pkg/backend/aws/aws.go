package aws

import (
	"context"
	"fmt"
	"net/url"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/pkg/errors"
	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/rclone/rclone/fs/config/configstruct"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/bringg/honey/pkg/place"
)

const Name = "aws"

var (
	log = logrus.WithField("backend", Name)
)

type (
	Backend struct {
		cls map[string]*ec2.Client
		opt Options
	}

	// Options defines the configuration for this backend
	Options struct {
		Region string `config:"region"`
	}

	ConcurrentSlice struct {
		sync.RWMutex
		Items []*place.Instance
	}
)

// Register with Backend
func init() {
	place.Register(&place.RegInfo{
		Name:        Name,
		Description: "Amazon EC2 Instances",
		NewBackend:  NewBackend,
		Options: []place.Option{
			{
				Name: "region",
				Help: "region name",
			},
		},
	})
}

func (cs *ConcurrentSlice) Append(item *place.Instance) {
	cs.Lock()
	defer cs.Unlock()

	cs.Items = append(cs.Items, item)
}

// NewBackend _
func NewBackend(ctx context.Context, m configmap.Mapper) (place.Backend, error) {
	// Parse config into Options struct
	opt := new(Options)
	err := configstruct.Set(m, opt)
	if err != nil {
		return nil, err
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "configuration error")
	}

	regions := []types.Region{{RegionName: aws.String(opt.Region)}}

	if opt.Region == "" {
		// get list of regions
		out, err := ec2.NewFromConfig(cfg.Copy()).
			DescribeRegions(ctx, &ec2.DescribeRegionsInput{})
		if err != nil {
			return nil, err
		}

		regions = out.Regions
	}

	cls := make(map[string]*ec2.Client, 0)
	for _, r := range func() []string {
		if opt.Region != "" {
			return []string{opt.Region}
		}

		var regs []string
		for _, reg := range regions {
			regs = append(regs, *reg.RegionName)
		}

		return regs
	}() {
		cfg := cfg.Copy()
		cfg.Region = r

		cls[r] = ec2.NewFromConfig(cfg)
	}

	return &Backend{
		cls: cls,
		opt: *opt,
	}, nil
}

func (b *Backend) Name() string {
	return Name
}

func (b *Backend) CacheKeyName(pattern string) string {
	return fmt.Sprintf("%s", pattern)
}

func (b *Backend) List(ctx context.Context, pattern string) (place.Printable, error) {
	input := &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []string{fmt.Sprintf("*%s*", pattern)},
			},
		},
	}

	instances := new(ConcurrentSlice)

	g, fCtx := errgroup.WithContext(ctx)

	for region, c := range b.cls {
		log.Debugf("using region %s", region)

		g.Go(func(c *ec2.Client) func() error {
			return func() error {
				result, err := c.DescribeInstances(fCtx, input)
				if err != nil {
					return err
				}

				for _, r := range result.Reservations {
					for _, instance := range r.Instances {
						// We need to see if the Name is one of the tags. It's not always
						// present and not required in Ec2.
						name := "None"
						for _, t := range instance.Tags {
							if *t.Key == "Name" {
								name = url.QueryEscape(*t.Value)
							}
						}

						instances.Append(&place.Instance{
							Model: place.Model{
								BackendName: Name,
								ID:          aws.ToString(instance.InstanceId),
								Name:        name,
								Type:        string(instance.InstanceType),
								Status:      aws.ToString((*string)(&instance.State.Name)),
								PrivateIP:   aws.ToString(instance.PrivateIpAddress),
								PublicIP:    aws.ToString(instance.PublicIpAddress),
							},
							Raw: instance,
						})
					}
				}

				return nil
			}
		}(c))
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return instances.Items, nil
}
