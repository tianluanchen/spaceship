package cmd

import (
	"os"
	"strings"

	"github.com/tianluanchen/spaceship/pkg"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "spaceship",
	Short: "Concurrent HTTP downloader, uploader client and server",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		level, _ := cmd.Flags().GetString("level")
		level = strings.ToUpper(level)
		switch level {
		case "DEBUG":
			pkg.SetLogLevel(pkg.LDEBUG)
		case "INFO":
			pkg.SetLogLevel(pkg.LINFO)
		case "WARN":
			pkg.SetLogLevel(pkg.LWARN)
		case "ERROR":
			pkg.SetLogLevel(pkg.LERROR)
		case "FATAL":
			pkg.SetLogLevel(pkg.LFATAL)
		}
		if name := cmd.Name(); name == "fetch" || name == "version" {
			return
		}
		if cmd.Name() != "conf" {
			for i := 0; i < len(args); i++ {
				args[i] = pkg.CleanPath(args[i])
			}
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
			logger.Debugln("read conf error:", err)
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
	rootCmd.PersistentFlags().String("level", "INFO", "log level, DEBUG INFO WARN ERROR FATAL")
}
