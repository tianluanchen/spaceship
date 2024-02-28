package cmd

import (
	"fmt"
	"time"

	"github.com/tianluanchen/spaceship/ship"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Ping server",
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
		n, _ := cmd.Flags().GetInt("count")
		interval, _ := cmd.Flags().GetDuration("interval")
		if interval < time.Millisecond*400 {
			logger.Fatal("interval too small, minimum 400ms")
		}
		fmt.Printf("Pinging %s :\n", client.GetPingURL())
		for i := 0; i < n || n < 0; i++ {
			if n < 0 {
				i = n - 2
			}
			if t, err := client.Ping(); err != nil {
				fmt.Printf("Unexpected error: %s\n", err)
			} else {
				fmt.Printf("Reply from %s : time=%s\n", client.GetPingURL(), t)
			}
			if i != n-1 {
				time.Sleep(interval)
			}
		}
	},
}

func init() {
	addCommonFlags(pingCmd)
	addSpacestationFlags(pingCmd)
	pingCmd.Flags().DurationP("interval", "i", time.Second, "time between sending each packet, minimum 400ms")
	pingCmd.Flags().IntP("count", "c", 3, "ping times, negative means infinity")
	rootCmd.AddCommand(pingCmd)
}
