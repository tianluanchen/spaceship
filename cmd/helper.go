package cmd

import (
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"spaceship/pkg"
	"spaceship/pkg/network"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

const (
	ConfFile               = ".spacestation"
	NameDisallowRedirects  = "disallow_redirects"
	NameProxyURL           = "proxy_url"
	NameInsecureSkipVerify = "insecure_skip_verify"
	NameAuthKeyHash        = "auth_key_hash"
	NameServerURL          = "server_url"
	NameResolveHostMap     = "resolve_host_map"
	NameCACertificate      = "ca_certificate"
)

var logger = pkg.NewLogger()

func newBar(max int64, options ...progressbar.Option) *progressbar.ProgressBar {
	options = append([]progressbar.Option{
		progressbar.OptionUseIECUnits(true),
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

func handleCACertificate(caPath string) *x509.CertPool {
	cert := viper.GetString(NameCACertificate)
	if caPath != "" {
		bs, err := os.ReadFile(caPath)
		if err != nil {
			logger.Fatalln(err)
		}
		cert = string(bs)
	}
	if cert == "" {
		return nil
	}
	certPool := x509.NewCertPool()
	ok := certPool.AppendCertsFromPEM([]byte(cert))
	if !ok {
		logger.Fatalln("failed to parse the certificate")
	}
	return certPool
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

func handleResolveHostMap(serverURL string, resolveArr ...string) map[string]string {
	u, err := pkg.ParseURL(serverURL)
	if err != nil {
		logger.Fatalln(err)
	}
	for _, v := range strings.Split(viper.GetString(NameResolveHostMap), ",") {
		if v != "" {
			resolveArr = append(resolveArr, "")
			copy(resolveArr[1:], resolveArr)
			resolveArr[0] = v
		}
	}
	resolveHostMap, err := parseResolveFlag(u.Hostname(), resolveArr...)
	if err != nil {
		logger.Fatalln(err)
	}
	return resolveHostMap
}

// add flag "http2"
func addTransportFlags(cmd *cobra.Command) {
	// 默认禁用http2是因为http2共用TCP连接，多路复用，对于并发下载大文件效率并没有HTTP/1.1高，
	// 此外可能出现"stream error: stream ID 5; INTERNAL_ERROR; received from peer" 的错误
	cmd.Flags().Bool("http2", false, "enable HTTP2 when uploading or downloading")
	cmd.Flags().IntP("concurrency", "c", 12, "number of concurrent goroutines")
}

// add flag "url" "auth" "no-redirect" "insecure" "proxy" "resolve" "cacert"
func addSpacestationFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("url", "u", "", "server url")
	cmd.Flags().StringP("auth", "a", "", "auth key")
	cmd.Flags().Bool("no-redirect", false, "disallow redirects")
	cmd.Flags().String("proxy", "", "proxy url")
	cmd.Flags().BoolP("insecure", "k", false, "insecure skip verify")
	cmd.Flags().StringArray("resolve", []string{}, "resolve host, * for all, eg. example.com:127.0.0.1  *:127.0.0.1")
	cmd.Flags().String("cacert", "", "CA certificate path")
}

// bind flag "url" "no-redirect" "insecure"
func bindSpacestationWithViper(cmd *cobra.Command) {
	viper.BindPFlag(NameProxyURL, cmd.Flags().Lookup("proxy"))
	viper.BindPFlag(NameServerURL, cmd.Flags().Lookup("url"))
	viper.BindPFlag(NameDisallowRedirects, cmd.Flags().Lookup("no-redirect"))
	viper.BindPFlag(NameInsecureSkipVerify, cmd.Flags().Lookup("insecure"))
}

func concat(v float64, unit string) string {
	integer := int64(v)
	unit = strings.TrimSuffix(unit, "B")
	if float64(integer) == float64(v) {
		return fmt.Sprintf("%d%s", integer, unit)
	} else {
		return fmt.Sprintf("%.1f%s", v, unit)
	}
}

func parseResolveFlag(hostname string, resolveArr ...string) (map[string]string, error) {
	resolveHostMap := make(map[string]string)
	for _, v := range resolveArr {
		ss := strings.Split(v, ":")
		err := errors.New("can't resolve host: " + v)
		if len(ss) == 1 {
			if network.IsIP(ss[0]) {
				resolveHostMap[hostname] = ss[0]
			} else {
				return nil, err
			}
		} else if len(ss) == 2 {
			if (network.IsDomain(ss[0]) || ss[0] == "*") && network.IsIP(ss[1]) {
				resolveHostMap[ss[0]] = ss[1]
			} else {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return resolveHostMap, nil
}

func parseHeader(headerArr ...string) http.Header {
	header := make(http.Header)
	for _, v := range headerArr {
		index := strings.Index(v, ":")
		if index > -1 {
			k := strings.Trim(v[0:index], "\r\n\t ")
			v := strings.Trim(v[index+1:], "\r\n\t ")
			if len(k) > 0 {
				header.Add(k, v)
			}
		}
	}
	return header
}
