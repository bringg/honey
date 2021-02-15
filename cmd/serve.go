package cmd

import (
	"github.com/spf13/cobra"

	"github.com/bringg/honey/pkg/resthttp"
)

var (
	serveCmd = &cobra.Command{
		Use:   "serve",
		Short: "Serve over a http protocol",
		RunE: func(command *cobra.Command, args []string) error {
			return resthttp.NewServer(&resthttp.Options{}).Serve()
		},
	}
)
