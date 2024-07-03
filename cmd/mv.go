package cmd

import (
	"strings"

	"spaceship/fetch"
	"spaceship/ship"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var mvCmd = &cobra.Command{
	Use:   "mv",
	Short: "Move remote file to specified path",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		remoteFile, newRemoteFile := args[0], args[1]
		if remoteFile == newRemoteFile {
			logger.Fatalln("same path")
		}
		var (
			serverURL  = viper.GetString(NameServerURL)
			proxyURL   = viper.GetString(NameProxyURL)
			insecure   = viper.GetBool(NameInsecureSkipVerify)
			noRedirect = viper.GetBool(NameDisallowRedirects)
		)
		auth, _ := cmd.Flags().GetString("auth")
		resolveArr, _ := cmd.Flags().GetStringArray("resolve")
		overwrite, _ := cmd.Flags().GetBool("overwrite")
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
		logger.Debugln("target url:", client.GetMoveFileURL(remoteFile, newRemoteFile, overwrite))
		logger.Debugf("insecure: %v  disallow redirects: %v proxy: %s", insecure, noRedirect, proxyURL)
		logger.Debugf("resolve host map: %v", resolveHostMap)
		logger.Debugf("specify CA certificate: %v", certPool != nil)
		if err := client.Move(remoteFile, newRemoteFile, overwrite); err == nil {
			logger.Infof("move %s to %s success", remoteFile, newRemoteFile)
		} else {
			logger.Fatalf("failed to move %s : %s", remoteFile, err)
		}
	},
}

func init() {
	addSpacestationFlags(mvCmd)
	mvCmd.Flags().Bool("overwrite", false, "if a file exists in the target path, overwrite")
	rootCmd.AddCommand(mvCmd)
}
