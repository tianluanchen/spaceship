package cmd

import (
	"crypto/x509"
	"spaceship/fetch"
	"spaceship/pkg"

	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

var fetchCmd = &cobra.Command{
	Use:     "fetch",
	Short:   "Concurrent download of web content to local",
	Example: "fetch <url> <output file?>",
	Args:    cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		u, err := pkg.ParseURL(args[0])
		if err != nil {
			logger.Fatalln(err)
		}
		args[0] = u.String()
		if len(args) == 1 {
			if name, err := pkg.ParseFileNameByURLPath(u.Path); err != nil {
				logger.Fatalf("unable to parse file name by %s, please specify <output file>", args[0])
			} else {
				logger.Debugf("parse file name %s by URL path", name)
				args = append(args, name)
			}
		}

		overwrite, _ := cmd.Flags().GetBool("overwrite")
		if info, err := os.Stat(args[1]); err == nil {
			if info.IsDir() {
				logger.Fatalf("%s is a directory", args[1])
			}
			if !overwrite {
				logger.Fatalf("%s already exists, you should use --overwrite", args[1])
			}
		}
		resolveArr, _ := cmd.Flags().GetStringArray("resolve")
		resolveHostMap, err := parseResolveFlag(u.Hostname(), resolveArr...)
		if err != nil {
			logger.Fatalln(err)
		}
		concurrency, _ := cmd.Flags().GetInt("concurrency")
		cookie, _ := cmd.Flags().GetString("cookie")
		insecure, _ := cmd.Flags().GetBool("insecure")
		noRedirect, _ := cmd.Flags().GetBool("no-redirect")
		proxyURL, _ := cmd.Flags().GetString("proxy")
		enableHTTP2, _ := cmd.Flags().GetBool("http2")
		headerArr, _ := cmd.Flags().GetStringArray("header")
		caPath, _ := cmd.Flags().GetString("cacert")
		var certPool *x509.CertPool
		if caPath != "" {
			certPool = handleCACertificate(caPath)
		}
		header := parseHeader(headerArr...)
		if header.Get("Cookie") == "" {
			header.Set("Cookie", cookie)
		}
		logger.Debugf("url: %s  insecure: %v  enable HTTP2: %v  disallow redirects: %v proxy: %s", args[0], insecure, enableHTTP2, noRedirect, proxyURL)
		logger.Debugf("header: %+v", header)
		logger.Debugf("resolve host map: %v", resolveHostMap)
		logger.Debugf("specify CA certificate: %v", certPool != nil)
		fetcher, err := fetch.NewFetcher(fetch.FetcherOption{
			InsecureSkipVerify: insecure,
			DisallowRedirects:  noRedirect,
			ProxyURL:           proxyURL,
			DisableHTTP2:       !enableHTTP2,
			ResolveHostMap:     resolveHostMap,
			RootCAs:            certPool,
			ResponsePreInspector: func(when int, resp *http.Response) error {
				if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
					bs := make([]byte, 256)
					n, _ := io.ReadAtLeast(resp.Body, bs, len(bs))
					lessBody := strings.Trim(string(bs[:n]), "\n\r\t ")
					return fmt.Errorf("unexpected status \"%s\" and front part of body is \"%s\"", resp.Status, lessBody)
				}
				return nil
			},
		})
		if err != nil {
			logger.Fatalln(err)
		}
		for key, value := range header {
			fetcher.Header.Del(key)
			for _, v := range value {
				fetcher.Header.Add(key, v)
			}
		}
		supported, length, err := fetcher.Inspect(args[0])
		if err != nil {
			logger.Fatalln(err)
		}
		if !supported {
			logger.Warnln("not support ranges")
		}
		fw, err := fetch.NewFileWriter(args[1])
		if err != nil {
			logger.Fatalln(err)
		}
		defer fw.Close()
		bar := newBar(length,
			progressbar.OptionSetDescription("Downloading [cyan]"+args[1]+"[reset]..."),
		)
		fw.OnWrite(func(n int, index int, start, end, length int64) {
			bar.Add(n)
		})
		if err := fetcher.DownloadWithManual(args[0], supported, length, &fetch.DownloadOption{
			Concurrency: concurrency,
			HookContext: fw.HookContext,
		}); err != nil {
			bar.Exit()
			fmt.Println()
			logger.Fatalln("download failed:", err.Error())
		}
		bar.Finish()
		fmt.Println()
		fw.Truncate(fw.WrittenN())
		logger.Infoln("download success")
	},
}

func init() {
	// 默认禁用http2是因为http2共用TCP连接，多路复用，对于并发下载大文件效率并没有HTTP/1.1高，
	// 此外可能出现"stream error: stream ID 5; INTERNAL_ERROR; received from peer" 的错误
	fetchCmd.Flags().Bool("http2", false, "enable HTTP2 when uploading or downloading")
	fetchCmd.Flags().Bool("no-redirect", false, "disallow redirects")
	fetchCmd.Flags().String("proxy", "", "proxy url")
	fetchCmd.Flags().StringArray("resolve", []string{}, "resolve host, * for all, eg. example.com:127.0.0.1  *:127.0.0.1")
	fetchCmd.Flags().BoolP("insecure", "k", false, "insecure skip verify")
	fetchCmd.Flags().String("cacert", "", "CA certificate path")
	fetchCmd.Flags().IntP("concurrency", "c", 12, "number of concurrent goroutines")
	fetchCmd.Flags().StringArrayP("header", "H", []string{}, "header, example: -H \"Cookie:a=1\"")
	fetchCmd.Flags().StringP("cookie", "C", "", "cookie, example: -C \"a=1\"")
	fetchCmd.Flags().Bool("overwrite", false, "overwrite")
	rootCmd.AddCommand(fetchCmd)
}
