package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/tianluanchen/spaceship/pkg"
	"github.com/tianluanchen/spaceship/ship"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

var fetchCmd = &cobra.Command{
	Use:     "fetch",
	Short:   "Concurrent download of web content to local",
	Example: "fetch <url> <output file?>",

	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 || len(args) > 2 {
			logger.Fatal("args length error, require <url> <output file?>")
		}
		if u, err := pkg.FixURL(args[0]); err != nil {
			logger.Fatal(err)
		} else {
			args[0] = u.String()
			if len(args) == 1 {
				if name, err := resolveFileNameFromURLPath(u.Path); err != nil {
					logger.Fatalf("unable to resolve file name from %s, require <output file>", args[0])
				} else {
					logger.Debugf("resolve file name %s from URL", name)
					args = append(args, name)
				}
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

		num, _ := cmd.Flags().GetInt("num")
		cookie, _ := cmd.Flags().GetString("cookie")
		headers, _ := cmd.Flags().GetStringArray("header")
		insecure, _ := cmd.Flags().GetBool("insecure")
		noRedirect, _ := cmd.Flags().GetBool("no-redirect")
		proxyURL, _ := cmd.Flags().GetString("proxy")

		h := http.Header{}
		for i := 0; i < len(headers); i++ {
			index := strings.Index(headers[i], ":")
			if index < 0 {
				continue
			}
			key := strings.TrimSpace(headers[i][:index])
			value := strings.TrimSpace(headers[i][index+1:])
			if key != "" && value != "" {
				h.Set(key, value)
			}
		}
		if h.Get("Cookie") == "" {
			h.Set("Cookie", cookie)
		}
		fetcher, err := ship.NewFetcher(ship.FetcherOption{
			InsecureSkipVerify: insecure,
			DisallowRedirects:  noRedirect,
			ProxyURL:           proxyURL,
			ResponsePreInspector: func(when int, resp *http.Response) error {
				if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
					bs := make([]byte, 128)
					n, _ := io.ReadAtLeast(resp.Body, bs, 128)
					lessBody := strings.Trim(string(bs[:n]), "\n\r\t ")
					return fmt.Errorf("unexpected status \"%s\" and front part of body is \"%s\"", resp.Status, lessBody)
				}
				return nil
			},
		})
		if err != nil {
			logger.Fatal(err)
			return
		}
		for key, value := range h {
			fetcher.Header.Del(key)
			for _, v := range value {
				fetcher.Header.Add(key, v)
			}
		}
		supported, length, err := fetcher.Inspect(args[0])
		if err != nil {
			logger.Fatal(err)
			return
		}
		if !supported {
			logger.Warn("not support ranges")
		}
		fw, err := ship.NewFileWriter(args[1])
		if err != nil {
			logger.Fatal(err)
			return
		}
		defer fw.Close()
		bar := newBar(length,
			progressbar.OptionSetDescription("Downloading [cyan]"+args[1]+"[reset]..."),
		)
		fw.OnWrite(func(n int, index int, start, end, length int64) {
			bar.Add(n)
		})
		if err := fetcher.DownloadWithManual(args[0], supported, length, &ship.FetcherDownloadOption{
			Concurrency: num,
			Handler:     fw.Handler,
		}); err != nil {
			bar.Exit()
			fmt.Println()
			logger.Fatal("download failed:", err.Error())
			return
		}
		bar.Finish()
		fmt.Println()
		fw.Truncate(fw.WrittenN())
		logger.Info("download success")
	},
}

func init() {
	addCommonFlags(fetchCmd)
	fetchCmd.Flags().IntP("num", "n", 6, "maximum number of concurrency")
	fetchCmd.Flags().StringArrayP("header", "H", []string{}, "header, example: -H \"Cookie:a=1\"")
	fetchCmd.Flags().StringP("cookie", "C", "", "cookie, example: -C \"a=1\"")
	fetchCmd.Flags().Bool("overwrite", false, "overwrite")

	rootCmd.AddCommand(fetchCmd)
}
