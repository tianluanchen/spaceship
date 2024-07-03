package cmd

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"spaceship/fetch"
	"spaceship/ship"

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
		n, _ := cmd.Flags().GetInt("count")
		auth, _ := cmd.Flags().GetString("auth")
		resolveArr, _ := cmd.Flags().GetStringArray("resolve")
		interval, _ := cmd.Flags().GetDuration("interval")
		caPath, _ := cmd.Flags().GetString("cacert")
		certPool := handleCACertificate(caPath)
		if interval < time.Millisecond*250 {
			logger.Fatalln("interval too small, minimum 400ms")
		}
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
		u := client.GetServerURL()
		var ip string
		if v, ok := resolveHostMap[u.Hostname()]; ok {
			ip = v
		} else if v, ok := resolveHostMap["*"]; ok {
			ip = v
		}
		if ip == "" {
			addrs, err := net.LookupHost(u.Hostname())
			if err != nil {
				logger.Fatalln(err)
			}
			ip = addrs[0]
		}
		port := u.Port()
		if port == "" {
			if u.Scheme == "https" {
				port = "443"
			} else {
				port = "80"
			}
		}
		pingURL := client.GetPingURL()
		logger.Debugf("insecure: %v  disallow redirects: %v proxy: %s", insecure, noRedirect, proxyURL)
		logger.Debugf("resolve host map: %v", resolveHostMap)
		logger.Debugf("specify CA certificate: %v", certPool != nil)
		fmt.Printf("Pinging %s (%s:%s) :\n", pingURL, ip, port)
		infinite := n <= 0
		i := 0
		var total, success int64
		var min, max, sum time.Duration
		printStats := func() {
			var avg time.Duration
			if success != 0 {
				avg = sum / time.Duration(success)
			}
			fmt.Printf("\nTotal = %d, Success = %d, Fail = %d, Pass Percentage = %.1f%%\nMin = %v, Max = %v, Avg = %v\n",
				total, success, total-success, float64(success)/float64(total)*100, min, max, avg)
		}
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT)
		go func() {
			<-sigChan
			printStats()
			os.Exit(0)
		}()
		defer printStats()
		for {
			t, err := client.Ping()
			if err != nil {
				fmt.Printf("Unexpected error: %s\n", err)
			} else {
				fmt.Printf("Reply from %s : time=%s\n", pingURL, t)
				success += 1
				sum += t
				if min == 0 || t < min {
					min = t
				}
				if t > max {
					max = t
				}
			}
			total += 1
			i++
			if infinite || i < n {
				time.Sleep(interval)
			} else {
				return
			}
		}
	},
}

func init() {
	addSpacestationFlags(pingCmd)
	pingCmd.Flags().DurationP("interval", "i", time.Second, "time between sending each packet, minimum 400ms")
	pingCmd.Flags().IntP("count", "c", 3, "ping times, nonpositive number means infinity")
	rootCmd.AddCommand(pingCmd)
}
