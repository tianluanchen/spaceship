package cmd

import (
	"fmt"
	"os"
	"path"

	"spaceship/fetch"
	"spaceship/ship"

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
		var (
			serverURL  = viper.GetString(NameServerURL)
			proxyURL   = viper.GetString(NameProxyURL)
			insecure   = viper.GetBool(NameInsecureSkipVerify)
			noRedirect = viper.GetBool(NameDisallowRedirects)
			remoteFile = args[0]
			localFile  = remoteFile
		)
		resolveArr, _ := cmd.Flags().GetStringArray("resolve")
		concurrency, _ := cmd.Flags().GetInt("concurrency")
		enableHTTP2, _ := cmd.Flags().GetBool("http2")
		overwrite, _ := cmd.Flags().GetBool("overwrite")
		auth, _ := cmd.Flags().GetString("auth")
		caPath, _ := cmd.Flags().GetString("cacert")
		certPool := handleCACertificate(caPath)
		resolveHostMap := handleResolveHostMap(serverURL, resolveArr...)
		client, err := ship.NewClient(ship.ClientOption{
			ServerURL: serverURL,
			FetcherOption: fetch.FetcherOption{
				InsecureSkipVerify: insecure,
				DisallowRedirects:  noRedirect,
				ProxyURL:           proxyURL,
				DisableHTTP2:       !enableHTTP2,
				ResolveHostMap:     resolveHostMap,
				RootCAs:            certPool,
			},
		})
		if err != nil {
			logger.Fatalln(err)
		}
		client.SetAuth(handleAuth(auth), true)
		if len(args) == 2 {
			localFile = args[1]
		}
		tempFile := localFile
		if info, err := os.Stat(localFile); err == nil {
			if info.IsDir() {
				logger.Fatalf("%s is a directory", localFile)
			}
			if !overwrite {
				logger.Fatalf("%s already exists, you should use --overwrite", localFile)
			}
			logger.Warnf("%s already exists, it will be overwritten after download finished", localFile)
			tempFile = path.Join(path.Dir(localFile), path.Base(localFile)+".temp")
			logger.Debugln("temp file:", tempFile)
		}

		var bar *progressbar.ProgressBar
		logger.Debugln("target url:", client.GetDownloadFileURL(remoteFile))
		logger.Debugf("councurrency: %d  overwrite: %v  insecure: %v  enable HTTP2: %v  disallow redirects: %v  proxy: %s", concurrency, overwrite, insecure, enableHTTP2, noRedirect, proxyURL)
		logger.Debugf("resolve host map: %v", resolveHostMap)
		logger.Debugf("specify CA certificate: %v", certPool != nil)
		if err := client.Get(concurrency, remoteFile, tempFile, func(beforeDownload bool, supported bool, length int64, n int) {
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
		if tempFile != localFile {
			if err := os.Rename(tempFile, localFile); err != nil {
				logger.Fatalf("rename %s to %s failed: %s", tempFile, localFile, err.Error())
			} else {
				logger.Debugf("rename %s to %s success", tempFile, localFile)
			}
		}
		logger.Infoln("download success")
	},
}

func init() {
	addSpacestationFlags(getCmd)
	addTransportFlags(getCmd)
	getCmd.Flags().Bool("overwrite", false, "if local file exists, overwrite")
	rootCmd.AddCommand(getCmd)
}
