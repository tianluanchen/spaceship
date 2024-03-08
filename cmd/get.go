package cmd

import (
	"fmt"
	"os"
	"path"

	"github.com/tianluanchen/spaceship/ship"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var getCmd = &cobra.Command{
	Use:     "get",
	Short:   "Concurrent download remote file to local",
	Example: "get <remote path> <local path?>",

	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 || len(args) > 2 {
			logger.Fatalln("args length error, require <path> <output file?>")
		}
		num, _ := cmd.Flags().GetInt("num")
		if num <= 0 {
			num = 1
		}
		var (
			serverURL  = viper.GetString(NameServerURL)
			proxyURL   = viper.GetString(NameProxyURL)
			insecure   = viper.GetBool(NameInsecureSkipVerify)
			noRedirect = viper.GetBool(NameDisallowRedirects)
			remoteFile = path.Clean(args[0])
			localFile  = remoteFile
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
			logger.Fatalln(err)
		}
		auth, _ := cmd.Flags().GetString("auth")
		client.SetAuth(handleAuth(auth), true)

		if len(args) == 2 {
			localFile = path.Clean(args[1])
		}
		overwrite, _ := cmd.Flags().GetBool("overwrite")

		if info, err := os.Stat(localFile); err == nil {
			if info.IsDir() {
				logger.Fatalf("%s is a directory", localFile)
			}
			if !overwrite {
				logger.Fatalf("%s already exists, you should use --overwrite", localFile)
			}
			logger.Warnf("%s already exists, it will be overwritten", localFile)
		}

		var bar *progressbar.ProgressBar
		logger.Debugln("target url:", client.GetDownloadFileURL(remoteFile))
		if err := client.Get(num, remoteFile, localFile, func(beforeDownload bool, supported bool, length int64, n int) {
			if beforeDownload {
				if !supported {
					logger.Warnln("not support ranges")
				}
				bar = newBar(length, progressbar.OptionSetDescription("Downloading [cyan]"+remoteFile+"[reset] to [green]"+localFile+"[reset]..."))
			} else {
				bar.Add(n)
			}
		}); err != nil {
			if bar != nil {
				bar.Exit()
				fmt.Println()
			}
			logger.Fatalln("download failed:", err.Error())
			return
		}
		bar.Clear()
		bar.Close()
		fmt.Println(bar.String())
		logger.Infoln("download success")
	},
}

func init() {
	addCommonFlags(getCmd)
	addSpacestationFlags(getCmd)
	getCmd.Flags().IntP("num", "n", 6, "maximum number of concurrency")
	getCmd.Flags().Bool("overwrite", false, "if local file exists, overwrite")
	rootCmd.AddCommand(getCmd)
}
