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

var putCmd = &cobra.Command{
	Use:     "put",
	Short:   "Concurrent upload local file to remote",
	Example: "put <local path> <remote path?> ",

	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 || len(args) > 2 {
			logger.Fatal("args length error, require <local path> <remote path?>")
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
			localFile  = path.Clean(args[0])
			remoteFile = localFile
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

		if len(args) == 2 {
			remoteFile = path.Clean(args[1])
		}
		if info, err := os.Stat(localFile); err == nil {
			if info.IsDir() {
				logger.Fatalf("%s is a directory", localFile)
			}
		}
		overwrite, _ := cmd.Flags().GetBool("overwrite")
		if overwrite {
			logger.Warn("if remote file or upload task exists, overwrite")
		}
		var bar *progressbar.ProgressBar
		logger.Debug("target url:", client.GetUploadFileURL(remoteFile))
		if err := client.Put(num, overwrite, localFile, remoteFile, func(beforeUpload bool, info ship.UploadInfo, n int) {
			if beforeUpload {
				logger.Debug("task id:", info.TaskID)
				bar = newBar(info.TotalSize, progressbar.OptionSetDescription("Uploading [cyan]"+localFile+"[reset] to [green]"+info.Path+"[reset]..."))
			} else {
				bar.Add(n)
			}
		}); err != nil {
			if bar != nil {
				bar.Exit()
				fmt.Println()
			}
			logger.Fatal("upload failed:", err.Error())
			return
		}
		bar.Clear()
		bar.Close()
		fmt.Println(bar.String())
		logger.Info("upload success")
	},
}

func init() {
	addCommonFlags(putCmd)
	addSpacestationFlags(putCmd)
	putCmd.Flags().IntP("num", "n", 6, "maximum number of concurrency")
	putCmd.Flags().Bool("overwrite", false, "if remote file or upload task exists, overwrite")
	rootCmd.AddCommand(putCmd)
}
