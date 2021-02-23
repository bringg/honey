package handlers

type (
	CustomBackend struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}

	BackendsResponse struct {
		Backends       []string        `json:"backends"`
		CustomBackends []CustomBackend `json:"custom_backends"`
	}
)
