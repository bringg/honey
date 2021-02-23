package k8s

import (
	"context"
	"fmt"
	"regexp"

	"github.com/pkg/errors"
	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/rclone/rclone/fs/config/configstruct"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/bringg/honey/pkg/place"
)

const Name = "k8s"

var (
	log = logrus.WithField("backend", Name)
)

type (
	Backend struct {
		client *kubernetes.Clientset
		opt    Options
	}

	Options struct {
		Context   string `config:"context"`
		Namespace string `config:"namespace"`
	}
)

// Register with Backend
func init() {
	place.Register(&place.RegInfo{
		Name:        Name,
		Description: "Kubernetes Pods",
		NewBackend:  NewBackend,
		Options: []place.Option{
			{
				Name:     "context",
				Help:     "k8s context",
				Required: true,
			},
			{
				Name:    "namespace",
				Help:    "k8s namespace",
				Default: metav1.NamespaceDefault,
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

	// Load new raw config
	kubeConfigOriginal, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).RawConfig()
	if err != nil {
		return nil, err
	}

	// We clone the config here to avoid changing the single loaded config
	kubeConfig := kubeConfigOriginal.DeepCopy()

	// If we should use a certain kube context use that
	activeContext := kubeConfig.CurrentContext

	// Set active context
	if opt.Context != "" && activeContext != opt.Context {
		activeContext = opt.Context
		kubeConfig.CurrentContext = opt.Context
	}

	if activeContext == "" {
		return nil, errors.New("k8s context is required")
	}

	log.Debugf("using context %s", activeContext)

	clientConfig := clientcmd.NewNonInteractiveClientConfig(*kubeConfig, activeContext, &clientcmd.ConfigOverrides{}, clientcmd.NewDefaultClientConfigLoadingRules())
	if kubeConfig.Contexts[activeContext] == nil {
		return nil, errors.Errorf("Error loading kube config, context '%s' doesn't exist", activeContext)
	}

	// Create new kube client
	cfg, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &Backend{
		client: clientset,
		opt:    *opt,
	}, nil
}

func (b *Backend) Name() string {
	return Name
}

func (b *Backend) CacheKeyName(pattern string) string {
	return fmt.Sprintf("%s-%s-%s", b.opt.Context, b.opt.Namespace, pattern)
}

func (b *Backend) List(ctx context.Context, pattern string) (place.Printable, error) {
	ns := ""
	if b.opt.Namespace != "" {
		ns = b.opt.Namespace
	}

	log.Debugf("using namespace: %s", ns)

	podFilter, err := regexp.Compile(fmt.Sprintf(".*%s.*", pattern))
	if err != nil {
		return nil, errors.Wrap(err, "failed to compile regular expression from query")
	}

	pods, err := b.client.
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
			Model: place.Model{
				BackendName: Name,
				ID:          string(pod.UID),
				Name:        pod.Name,
				Type:        "pod",
				Status:      string(pod.Status.Phase),
				PrivateIP:   pod.Status.PodIP,
				PublicIP:    pod.Status.HostIP,
			},
			Raw: pod,
		})
	}

	return instances, nil
}
