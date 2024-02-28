package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/tianluanchen/spaceship/pkg"
	"github.com/tianluanchen/spaceship/ship"

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
		logger.Debug("target url:", client.GetListURL())
		utc, _ := cmd.Flags().GetBool("utc")
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
			fmt.Printf("%s %7s  %s\n", formattedTime, pkg.FormatSize(info.Size, func(v float64, unit string) string {
				integer := int64(v)
				unit = strings.TrimSuffix(unit, "B")
				if float64(integer) == float64(v) {
					return fmt.Sprintf("%d%s", integer, unit)
				} else {
					return fmt.Sprintf("%.1f%s", v, unit)
				}
			}), info.Name)

		}); err != nil {
			logger.Fatal(err)
		}
	},
}

func init() {
	addCommonFlags(lsCmd)
	addSpacestationFlags(lsCmd)
	lsCmd.Flags().Bool("utc", false, "print modified time in utc")
	rootCmd.AddCommand(lsCmd)
}
