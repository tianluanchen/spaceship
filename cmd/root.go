package cmd

import (
	"os"

	"github.com/tianluanchen/spaceship/pkg"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var logger = pkg.NewLogger(nil)

var rootCmd = &cobra.Command{
	Use:   "spaceship",
	Short: "Concurrent HTTP downloader, uploader client and server",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {

		debug, _ := cmd.Flags().GetBool("debug")
		if debug {
			logger.SetDebugLevel()
		} else {
			logger.SetInfoLevel()
		}
		if cmd.Name() == "fetch" {
			return
		}
		for i := 0; i < len(args); i++ {
			args[i] = pkg.CleanPath(args[i])
		}

		viper.SetDefault(NameServerURL, "")
		viper.SetDefault(NameProxyURL, "")
		viper.SetDefault(NameAuthKeyHash, "")
		viper.SetDefault(NameDisallowRedirects, false)
		viper.SetDefault(NameInsecureSkipVerify, false)

		viper.SetConfigType("json")
		viper.SetConfigName(ConfFile)
		viper.AddConfigPath("$HOME")

		if err := viper.ReadInConfig(); err != nil {
			logger.Debug("read conf error:", err)
		}

		bindSpacestationWithViper(cmd)

	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "debug mode")
}
