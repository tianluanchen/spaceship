package cmd

import (
	"fmt"
	"time"

	"spaceship/fetch"
	"spaceship/pkg"
	"spaceship/ship"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List remote files",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		var (
			serverURL  = viper.GetString(NameServerURL)
			proxyURL   = viper.GetString(NameProxyURL)
			insecure   = viper.GetBool(NameInsecureSkipVerify)
			noRedirect = viper.GetBool(NameDisallowRedirects)
		)
		utc, _ := cmd.Flags().GetBool("utc")
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
		logger.Debugln("target url:", client.GetListURL())
		logger.Debugf("utc: %v  insecure: %v  disallow redirects: %v  proxy: %s", utc, insecure, noRedirect, proxyURL)
		logger.Debugf("resolve host map: %v", resolveHostMap)
		logger.Debugf("specify CA certificate: %v", certPool != nil)
		if err := client.List(func(info *ship.FileInfo) {
			t := time.Unix(info.ModTime, 0)
			var formattedTime string
			if utc {
				_, offset := t.Zone()
				t = t.Add(-1 * time.Duration(offset) * time.Second)
				formattedTime = t.Format("2006-01-02 15:04:05")
				formattedTime += " UTC"
			} else {
				formattedTime = t.Format("2006-01-02 15:04:05")
			}
			fmt.Printf("%s %7s  %s\n", formattedTime, pkg.FormatSize(info.Size, concat), info.Name)

		}); err != nil {
			logger.Fatalln(err)
		}
	},
}

func init() {
	addSpacestationFlags(lsCmd)
	lsCmd.Flags().Bool("utc", false, "print modified time in utc")
	rootCmd.AddCommand(lsCmd)
}
