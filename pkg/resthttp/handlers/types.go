package handlers

type (
	Backend struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		Type string `json:"type"`
	}

	BackendsResponse struct {
		Data []Backend `json:"data"`
	}
)
