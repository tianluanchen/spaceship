package cmd

import (
	"path"
	"strings"

	"github.com/tianluanchen/spaceship/ship"

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

		client, err := ship.NewClient(ship.ClientOption{
			ServerURL: serverURL,
			FetcherOption: ship.FetcherOption{
				InsecureSkipVerify: insecure,
				DisallowRedirects:  noRedirect,
				ProxyURL:           proxyURL,
			},
		})
		if err != nil {
			logger.Fatal(err)
		}
		auth, _ := cmd.Flags().GetString("auth")
		client.SetAuth(handleAuth(auth), true)
		if strings.TrimSpace(args[0]) == "" {
			logger.Fatal("empty path")
		}
		remoteFile := path.Clean(args[0])
		logger.Debug("target url:", client.GetDeleteFileURL(remoteFile))
		if err := client.Delete(remoteFile); err == nil {
			logger.Infof("delete %s success", remoteFile)
		} else {
			logger.Fatalf("failed to delete %s : %s", remoteFile, err)
		}
	},
}

func init() {
	addCommonFlags(rmCmd)
	addSpacestationFlags(rmCmd)
	rootCmd.AddCommand(rmCmd)
}
