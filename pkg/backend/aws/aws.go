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

	"github.com/shareed2k/honey/pkg/place"
)

const Name = "aws"

type (
	Backend struct {
		cls map[string]*ec2.Client
		opt Options
	}

	// Options defines the configuration for this backend
	Options struct {
		Region string `config:"region"`
	}
)

// Register with Backend
func init() {
	place.Register(&place.RegInfo{
		Name:       Name,
		NewBackend: NewBackend,
	})
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

func worker(ctx context.Context, wg *sync.WaitGroup, c *ec2.Client, input *ec2.DescribeInstancesInput, region string, instances *[]*place.Instance) {
	defer wg.Done()

	result, err := c.DescribeInstances(ctx, input)
	if err != nil {
		fmt.Println("err: ", err)
		return
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

			*instances = append(*instances, &place.Instance{
				BackendName: Name,
				ID:          aws.ToString(instance.InstanceId),
				Name:        name,
				Type:        string(instance.InstanceType),
				//Status:    instance,
				PrivateIP: aws.ToString(instance.PrivateIpAddress),
				PublicIP:  aws.ToString(instance.PublicIpAddress),
			})
		}
	}
}

func (b *Backend) Name() string {
	return Name
}

func (b *Backend) List(ctx context.Context, pattern string) ([]*place.Instance, error) {
	input := &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []string{fmt.Sprintf("*%s*", pattern)},
			},
			/* {
				Name:   aws.String("instance-state-name"),
				Values: []string{"running", "pending"},
			}, */
		},
	}

	var wg sync.WaitGroup
	instanses := make([]*place.Instance, 0)
	for region, c := range b.cls {
		wg.Add(1)
		go worker(ctx, &wg, c, input, region, &instanses)
	}

	wg.Wait()

	return instanses, nil
}
