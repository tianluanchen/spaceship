package cmd

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/tianluanchen/spaceship/pkg"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

const (
	ConfFile               = ".spacestation"
	NameDisallowRedirects  = "DisallowRedirects"
	NameProxyURL           = "ProxyURL"
	NameInsecureSkipVerify = "InsecureSkipVerify"
	NameAuthKeyHash        = "AuthKeyHash"
	NameServerURL          = "ServerURL"
)

var logger = pkg.NewLogger()

func newBar(max int64, options ...progressbar.Option) *progressbar.ProgressBar {
	options = append([]progressbar.Option{
		progressbar.OptionSpinnerType(14),
		progressbar.OptionUseANSICodes(true),
		progressbar.OptionSetPredictTime(true),
		progressbar.OptionShowElapsedTimeOnFinish(),
		progressbar.OptionShowBytes(true),
		progressbar.OptionThrottle(time.Millisecond * 100),
		progressbar.OptionShowCount(),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetRenderBlankState(true),
	}, options...)
	return progressbar.NewOptions64(max, options...)
}

func resolveFileNameFromURLPath(p string) (string, error) {
	pathSlice := strings.Split(p, "/")
	name := ""
	if len(pathSlice) > 0 && pathSlice[len(pathSlice)-1] != "" {
		name = path.Clean(pathSlice[len(pathSlice)-1])
	}
	if name == "" {
		return "", fmt.Errorf("unable to resolve file name from %s", p)
	}
	return name, nil
}

func handleAuth(auth string) (authHash string) {
	authHash = viper.GetString(NameAuthKeyHash)
	if auth == "" && authHash == "" {
		fmt.Printf("Please input auth key: ")
		password, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			logger.Fatalln(err)
		}
		auth = string(password)
		for i := 0; i < len(auth); i++ {
			fmt.Printf("*")
		}
		fmt.Println()
	}
	if auth != "" {
		authHash = pkg.CalculateSHA256(auth)
	}
	return
}

// add flag "no-redirect" "insecure" "proxy"
func addCommonFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("no-redirect", false, "disallow redirects")
	cmd.Flags().String("proxy", "", "proxy url")
	cmd.Flags().Bool("insecure", false, "insecure skip verify")
}

// add flag "url" "auth"
func addSpacestationFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("url", "u", "", "server url")
	cmd.Flags().StringP("auth", "a", "", "auth key")
}

// bind flag "url" "no-redirect" "insecure"
func bindSpacestationWithViper(cmd *cobra.Command) {
	viper.BindPFlag(NameProxyURL, cmd.Flags().Lookup("proxy"))
	viper.BindPFlag(NameServerURL, cmd.Flags().Lookup("url"))
	viper.BindPFlag(NameDisallowRedirects, cmd.Flags().Lookup("no-redirect"))
	viper.BindPFlag(NameInsecureSkipVerify, cmd.Flags().Lookup("insecure"))
}
