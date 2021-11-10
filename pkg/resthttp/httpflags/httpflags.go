package httpflags

import (
	"github.com/rclone/rclone/fs/config/flags"
	"github.com/spf13/pflag"

	"github.com/bringg/honey/pkg/resthttp"
)

// Options set by command line flags
var (
	Opt = resthttp.DefaultOpt
)

// AddFlagsPrefix adds flags for the resthttp
func AddFlagsPrefix(flagSet *pflag.FlagSet, opt *resthttp.Options) {
	flags.StringVarP(flagSet, &opt.ListenAddr, "addr", "", opt.ListenAddr, "IPaddress:Port or :Port to bind server to.")
	flags.DurationVarP(flagSet, &opt.ServerReadTimeout, "server-read-timeout", "", opt.ServerReadTimeout, "Timeout for server reading data")
	flags.DurationVarP(flagSet, &opt.ServerWriteTimeout, "server-write-timeout", "", opt.ServerWriteTimeout, "Timeout for server writing data")
	flags.IntVarP(flagSet, &opt.MaxHeaderBytes, "max-header-bytes", "", opt.MaxHeaderBytes, "Maximum size of request header")
	flags.StringVarP(flagSet, &opt.SslCert, "cert", "", opt.SslCert, "SSL PEM key (concatenation of certificate and CA certificate)")
	flags.StringVarP(flagSet, &opt.SslKey, "key", "", opt.SslKey, "SSL PEM Private key")
	flags.StringVarP(flagSet, &opt.ClientCA, "client-ca", "", opt.ClientCA, "Client certificate authority to verify clients with")
	flags.StringVarP(flagSet, &opt.Realm, "realm", "", opt.Realm, "realm for authentication")
	flags.StringVarP(flagSet, &opt.BasicUser, "user", "", opt.BasicUser, "User name for authentication.")
	flags.StringVarP(flagSet, &opt.BasicPass, "pass", "", opt.BasicPass, "Password for authentication.")
	flags.StringVarP(flagSet, &opt.BaseURL, "baseurl", "", opt.BaseURL, "Prefix for URLs - leave blank for root.")
	flags.BoolVarP(flagSet, &opt.UI, "ui", "", opt.UI, "start web UI")
}

// AddFlags adds flags for the resthttp
func AddFlags(flagSet *pflag.FlagSet) {
	AddFlagsPrefix(flagSet, &Opt)
}
