package k8s

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/rclone/rclone/fs/config/configstruct"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/shareed2k/honey/pkg/place"
)

const Name = "k8s"

type (
	Backend struct {
		c   *kubernetes.Clientset
		opt Options
	}

	Options struct {
		Context   string `config:"context"`
		Namespace string `config:"namespace"`
	}
)

// Register with Backend
func init() {
	place.Register(&place.RegInfo{
		Name:       Name,
		NewBackend: NewBackend,
		Options: []place.Option{
			{
				Name: "context",
				Help: "k8s context",
			},
			{
				Name: "namespace",
				Help: "k8s namespace",
			},
		},
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

	home, err := homedir.Dir()
	if err != nil {
		return nil, err
	}

	// use the current context in kubeconfig
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	loadingRules.ExplicitPath = filepath.Join(home, ".kube", "config")

	var configOverrides *clientcmd.ConfigOverrides
	if opt.Context != "" {
		configOverrides = &clientcmd.ConfigOverrides{
			ClusterDefaults: clientcmd.ClusterDefaults,
			CurrentContext:  opt.Context,
		}
	}

	k8sCfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides).ClientConfig()
	if err != nil {
		return nil, err
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(k8sCfg)
	if err != nil {
		return nil, err
	}

	return &Backend{
		c:   clientset,
		opt: *opt,
	}, nil
}

func (b *Backend) Name() string {
	return Name
}

func (b *Backend) List(ctx context.Context, pattern string) (place.Printable, error) {
	ns := ""
	if b.opt.Namespace != "" {
		ns = b.opt.Namespace
	}

	podFilter, err := regexp.Compile(fmt.Sprintf(".*%s.*", pattern))
	if err != nil {
		return nil, errors.Wrap(err, "failed to compile regular expression from query")
	}

	pods, err := b.c.
		CoreV1().
		Pods(ns).
		List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	instances := make([]*place.Instance, 0)
	for _, pod := range pods.Items {
		if !podFilter.MatchString(pod.Name) {
			continue
		}

		instances = append(instances, &place.Instance{
			BackendName: Name,
			ID:          string(pod.UID),
			Name:        pod.Name,
			Type:        "pod",
			Status:      string(pod.Status.Phase),
			PrivateIP:   pod.Status.PodIP,
			PublicIP:    pod.Status.HostIP,
		})
	}

	return instances, nil
}
