package cmd

import (
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"spaceship/pkg"
	"strings"
	"time"

	"github.com/klauspost/compress/zip"
	"github.com/spf13/cobra"
)

var unzipCmd = &cobra.Command{
	Use:   "unzip",
	Short: "Unarchive zip",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		concat := func(v float64, unit string) string {
			integer := int64(v)
			unit = strings.TrimSuffix(unit, "B")
			if float64(integer) == float64(v) {
				return fmt.Sprintf("%d%s", integer, unit)
			} else {
				return fmt.Sprintf("%.1f%s", v, unit)
			}
		}
		list, _ := cmd.Flags().GetBool("list")
		if list && len(args) > 1 {
			logger.Fatalln("only one zip can be listed at a time")
		}
		zipReader, err := zip.OpenReader(args[0])
		if err != nil {
			logger.Fatalln(err)
		}
		defer zipReader.Close()

		if list {
			for _, f := range zipReader.File {
				fmt.Printf("%s\t%s\t%s\t%s\n",
					f.Mode(),
					pkg.FormatSize(f.UncompressedSize64, concat),
					f.Modified.Format("2006-01-02 15:04:05"),
					f.Name,
				)
			}
			return
		}
		root, _ := cmd.Flags().GetString("root")
		all, _ := cmd.Flags().GetBool("all")
		overwrite, _ := cmd.Flags().GetBool("overwrite")
		rule, _ := cmd.Flags().GetString("exclude")
		excludeRegexp, err := regexp.Compile(rule)
		if err != nil {
			logger.Fatalln(err)
		}
		root = path.Clean(strings.ReplaceAll(root, `\`, `/`))
		if info, err := os.Stat(root); err != nil {
			logger.Fatalln(err)
		} else if !info.IsDir() {
			logger.Fatalln(root, " is not a directory")
		}
		if all {
			logger.Debugln("unarchive all files, the regexp to exclude was ignored")
		} else {
			logger.Debugln("regexp to exclude:", excludeRegexp.String())
		}
		start := time.Now()
		for _, f := range zipReader.File {
			p := path.Join(root, path.Clean(strings.TrimLeft(strings.ReplaceAll(f.Name, `\`, `/`), "/")))
			mode := f.Mode()
			if !all && (excludeRegexp.MatchString(p) || excludeRegexp.MatchString(path.Base(p))) {
				logger.Warnln("skip", p, "because it matched regexp to exclude")
				continue
			}
			if pkg.GetLogLevel() <= pkg.LINFO {
				fmt.Printf("%s\t%s\t%s\n",
					mode,
					pkg.FormatSize(f.UncompressedSize64, concat),
					p,
				)
			}
			existDir := false
			if info, err := os.Stat(p); err != nil && !os.IsNotExist(err) {
				logger.Fatalln(err)
			} else if err == nil {
				if info.IsDir() {
					existDir = true
					if !mode.IsDir() {
						logger.Fatalln(p, "is a directory, you should delete it manually")
					}
				} else if !overwrite {
					logger.Fatalln(p, "already exists, you should use --overwrite")
				}

			}
			if mode.IsDir() {
				if !existDir {
					if err := os.MkdirAll(p, f.Mode()); err != nil {
						logger.Fatalln(err)
					}
				}
			} else {
				fr, err := f.Open()
				if err != nil {
					logger.Fatalln(err)
				}
				fw, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
				if err != nil {
					fr.Close()
					logger.Fatalln(err)
				}
				_, err = io.Copy(fw, fr)
				fr.Close()
				fw.Close()
				if err != nil {
					logger.Fatalln(err)
				}
			}
		}
		end := time.Now()
		logger.Warnln("total time:", end.Sub(start))
	},
}

func init() {
	unzipCmd.Flags().Bool("overwrite", false, "overwrite output file")
	unzipCmd.Flags().BoolP("list", "l", false, "list files of the specified zip archive")
	unzipCmd.Flags().Bool("all", false, "unarchive all files, the regexp to exclude files will be ignored")
	unzipCmd.Flags().String("exclude", `node_modules|__pycache__|venv|\.git`, "specify regexp to exclude files, first match the basename, then match the archive path")
	unzipCmd.Flags().String("root", "./", "the root directory to unarchive")
	rootCmd.AddCommand(unzipCmd)
}
