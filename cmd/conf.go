package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var confCmd = &cobra.Command{
	Use:     "conf",
	Short:   "Read and set config",
	Example: fmt.Sprintf("  write: conf -w %s 127.0.0.1:8080\n  unset: conf -u %s\n", NameServerURL, NameServerURL),
	Run: func(cmd *cobra.Command, args []string) {
		rm, _ := cmd.Flags().GetBool("rm")
		unset, _ := cmd.Flags().GetBool("unset")
		write, _ := cmd.Flags().GetBool("write")
		where, _ := cmd.Flags().GetBool("where")
		writeAuth, _ := cmd.Flags().GetBool("write-auth")
		writeCA, _ := cmd.Flags().GetBool("write-ca")
		if !writeAuth && !writeCA && !rm && !unset && !write && !where {
			if len(args) > 0 {
				for _, k := range args {
					if !viper.IsSet(k) {
						fmt.Println()
					} else {
						fmt.Println(viper.Get(k))
					}
				}
				return
			}
			for k, v := range viper.AllSettings() {
				fmt.Println(k + "=" + fmt.Sprint(v))
			}
			return
		}
		filename := viper.ConfigFileUsed()
		if where {
			fmt.Println(filename)
			return
		}

		if rm {
			if filename == "" {
				logger.Fatalln("no used config file")
			}
			if err := os.Remove(filename); err != nil {
				logger.Fatalln(err)
			} else {
				logger.Infof("remove config file %s successfully", filename)
			}
			return
		}
		if writeAuth {
			if len(args) != 0 {
				logger.Fatalln("not need args")
			}
			viper.Set(NameAuthKeyHash, "")
			viper.Set(NameAuthKeyHash, handleAuth(""))
		} else if writeCA {
			if len(args) != 1 {
				logger.Fatalln("args length error, need <ca_file>")
			}
			bs, err := os.ReadFile(args[0])
			if err != nil {
				logger.Fatalln(err)
			}
			viper.Set(NameCACertificate, string(bs))
		} else if write {
			if len(args) != 2 {
				logger.Fatalln("args length error, need <key> <value>")
			}
			switch strings.ToLower(args[0]) {
			case strings.ToLower(NameDisallowRedirects), strings.ToLower(NameInsecureSkipVerify):
				v := strings.ToLower(args[1])
				if v == "true" || v == "false" {
					viper.Set(args[0], v == "true")
				} else {
					logger.Fatalln("need true or false")
				}
			default:
				viper.Set(args[0], args[1])
			}
		} else if unset {
			if len(args) == 0 {
				logger.Fatalln("args length error, need <key> ...")
			}
			configMap := viper.AllSettings()

			for _, k := range args {
				delete(configMap, strings.ToLower(k))
			}
			bs, _ := json.MarshalIndent(configMap, "", " ")
			if err := viper.ReadConfig(bytes.NewReader(bs)); err != nil {
				logger.Fatalln(err)
			}
		}
		if err := viper.WriteConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				err = viper.SafeWriteConfig()
			}
			if err != nil {
				logger.Fatalln(err)
			}
		}
	},
}

func init() {
	confCmd.Flags().BoolP("unset", "u", false, "unset config")
	confCmd.Flags().BoolP("write", "w", false, "write config")
	confCmd.Flags().Bool("rm", false, "remove the conf file")
	confCmd.Flags().Bool("where", false, "print current conf file path")
	confCmd.Flags().Bool("write-auth", false, "write auth key")
	confCmd.Flags().Bool("write-ca", false, "write CA certificate by reading file")
	confCmd.MarkFlagsMutuallyExclusive("write-auth", "unset", "write", "rm", "where")
	rootCmd.AddCommand(confCmd)
}
