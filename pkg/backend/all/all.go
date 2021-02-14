package all

import (
	// part of registry
	_ "github.com/bringg/honey/pkg/backend/aws"
	_ "github.com/bringg/honey/pkg/backend/consul"
	_ "github.com/bringg/honey/pkg/backend/gcp"
	_ "github.com/bringg/honey/pkg/backend/k8s"
)
