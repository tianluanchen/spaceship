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

var putCmd = &cobra.Command{
	Use:     "put",
	Short:   "Concurrent upload local file to remote",
	Example: "put <local path> <remote path?> ",

	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 || len(args) > 2 {
			logger.Fatalln("args length error, require <local path> <remote path?>")
		}
		var (
			serverURL  = viper.GetString(NameServerURL)
			proxyURL   = viper.GetString(NameProxyURL)
			insecure   = viper.GetBool(NameInsecureSkipVerify)
			noRedirect = viper.GetBool(NameDisallowRedirects)
			localFile  = args[0]
			remoteFile = path.Base(localFile)
		)
		auth, _ := cmd.Flags().GetString("auth")
		resolveArr, _ := cmd.Flags().GetStringArray("resolve")
		overwrite, _ := cmd.Flags().GetBool("overwrite")
		concurrency, _ := cmd.Flags().GetInt("concurrency")
		enableHTTP2, _ := cmd.Flags().GetBool("http2")
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
			remoteFile = args[1]
		}
		if info, err := os.Stat(localFile); err == nil {
			if info.IsDir() {
				logger.Fatalf("%s is a directory", localFile)
			}
		}
		if overwrite {
			logger.Warnln("if remote file or upload task exists, overwrite")
		}
		var bar *progressbar.ProgressBar
		logger.Debugln("target url:", client.GetUploadFileURL(remoteFile))
		logger.Debugf("councurrency: %d  overwrite: %v  insecure: %v  enable HTTP2: %v  disallow redirects: %v  proxy: %s", concurrency, overwrite, insecure, enableHTTP2, noRedirect, proxyURL)
		logger.Debugf("resolve host map: %v", resolveHostMap)
		logger.Debugf("specify CA certificate: %v", certPool != nil)
		if err := client.Put(concurrency, overwrite, localFile, remoteFile, func(beforeUpload bool, info ship.UploadInfo, n int) {
			if beforeUpload {
				logger.Debugln("task id:", info.TaskID)
				bar = newBar(info.TotalSize, progressbar.OptionSetDescription("Uploading [cyan]"+localFile+"[reset] to [green]"+info.Path+"[reset]..."))
			} else {
				bar.Add(n)
			}
		}); err != nil {
			if bar != nil {
				bar.Exit()
				fmt.Println()
			}
			logger.Fatalln("upload failed:", err.Error())
			return
		}
		bar.Clear()
		bar.Close()
		fmt.Println(bar.String())
		logger.Infoln("upload success")
	},
}

func init() {
	addSpacestationFlags(putCmd)
	addTransportFlags(putCmd)
	putCmd.Flags().Bool("overwrite", false, "if remote file or upload task exists, overwrite")
	rootCmd.AddCommand(putCmd)
}
