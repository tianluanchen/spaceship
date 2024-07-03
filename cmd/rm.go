package cmd

import (
	"strings"

	"spaceship/fetch"
	"spaceship/ship"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rmCmd = &cobra.Command{
	Use:   "rm",
	Short: "Remove a remote file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var (
			serverURL  = viper.GetString(NameServerURL)
			proxyURL   = viper.GetString(NameProxyURL)
			insecure   = viper.GetBool(NameInsecureSkipVerify)
			noRedirect = viper.GetBool(NameDisallowRedirects)
		)
		auth, _ := cmd.Flags().GetString("auth")
		resolveArr, _ := cmd.Flags().GetStringArray("resolve")
		caPath, _ := cmd.Flags().GetString("cacert")
		certPool := handleCACertificate(caPath)
		resolveHostMap := handleResolveHostMap(serverURL, resolveArr...)
		client, err := ship.NewClient(ship.ClientOption{
			ServerURL: serverURL,
			FetcherOption: fetch.FetcherOption{
				InsecureSkipVerify: insecure,
				DisallowRedirects:  noRedirect,
				ProxyURL:           proxyURL,
				ResolveHostMap:     resolveHostMap,
				RootCAs:            certPool,
			},
		})
		if err != nil {
			logger.Fatalln(err)
		}
		client.SetAuth(handleAuth(auth), true)
		if strings.TrimSpace(args[0]) == "" {
			logger.Fatalln("empty path")
		}
		remoteFile := args[0]
		logger.Debugln("target url:", client.GetDeleteFileURL(remoteFile))
		logger.Debugf("insecure: %v  disallow redirects: %v proxy: %s", insecure, noRedirect, proxyURL)
		logger.Debugf("resolve host map: %v", resolveHostMap)
		logger.Debugf("specify CA certificate: %v", certPool != nil)
		if err := client.Delete(remoteFile); err == nil {
			logger.Infof("delete %s success", remoteFile)
		} else {
			logger.Fatalf("failed to delete %s : %s", remoteFile, err)
		}
	},
}

func init() {
	addSpacestationFlags(rmCmd)
	rootCmd.AddCommand(rmCmd)
}
