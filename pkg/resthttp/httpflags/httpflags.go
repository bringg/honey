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
func AddFlagsPrefix(flagSet *pflag.FlagSet, Opt *resthttp.Options) {
	flags.StringVarP(flagSet, &Opt.ListenAddr, "addr", "", Opt.ListenAddr, "IPaddress:Port or :Port to bind server to.")
	flags.DurationVarP(flagSet, &Opt.ServerReadTimeout, "server-read-timeout", "", Opt.ServerReadTimeout, "Timeout for server reading data")
	flags.DurationVarP(flagSet, &Opt.ServerWriteTimeout, "server-write-timeout", "", Opt.ServerWriteTimeout, "Timeout for server writing data")
	flags.IntVarP(flagSet, &Opt.MaxHeaderBytes, "max-header-bytes", "", Opt.MaxHeaderBytes, "Maximum size of request header")
	flags.StringVarP(flagSet, &Opt.SslCert, "cert", "", Opt.SslCert, "SSL PEM key (concatenation of certificate and CA certificate)")
	flags.StringVarP(flagSet, &Opt.SslKey, "key", "", Opt.SslKey, "SSL PEM Private key")
	flags.StringVarP(flagSet, &Opt.ClientCA, "client-ca", "", Opt.ClientCA, "Client certificate authority to verify clients with")
	flags.StringVarP(flagSet, &Opt.Realm, "realm", "", Opt.Realm, "realm for authentication")
	flags.StringVarP(flagSet, &Opt.BasicUser, "user", "", Opt.BasicUser, "User name for authentication.")
	flags.StringVarP(flagSet, &Opt.BasicPass, "pass", "", Opt.BasicPass, "Password for authentication.")
	flags.StringVarP(flagSet, &Opt.BaseURL, "baseurl", "", Opt.BaseURL, "Prefix for URLs - leave blank for root.")
}

// AddFlags adds flags for the resthttp
func AddFlags(flagSet *pflag.FlagSet) {
	AddFlagsPrefix(flagSet, &Opt)
}
