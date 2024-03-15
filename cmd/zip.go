package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/klauspost/compress/zip"
	"github.com/spf13/cobra"
	"github.com/tianluanchen/spaceship/pkg"
)

var zipCmd = &cobra.Command{
	Use:   "zip",
	Short: "archive files with zip and list archive files from zip",
	Long:  "archive files with zip, set log level to WARN to get better performance",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		list, _ := cmd.Flags().GetBool("list")
		if list {
			if len(args) > 1 {
				logger.Fatalln("only list one zip archive at a time")
			}
			r, err := zip.OpenReader(args[0])
			if err != nil {
				logger.Fatalln(err)
			}
			defer r.Close()
			for _, f := range r.File {
				fmt.Printf("%s\t%s\t%s\t%s\n",
					f.Mode(),
					pkg.FormatSize(f.UncompressedSize64, concat),
					f.Modified.Format("2006-01-02 15:04:05"),
					f.Name,
				)
			}
			return
		}

		glob, _ := cmd.Flags().GetBool("glob")
		output, _ := cmd.Flags().GetString("output")
		all, _ := cmd.Flags().GetBool("all")
		overwrite, _ := cmd.Flags().GetBool("overwrite")
		rule, _ := cmd.Flags().GetString("exclude")
		if rule == "" {
			rule = `^(node_modules|__pycache__|venv|\.git)$`
		}
		excludeRegexp, err := regexp.Compile(rule)
		if err != nil {
			logger.Fatalln(err)
		}
		if all {
			logger.Debugln("archive all files except output file, the regexp to exclude was ignored")
		} else {
			logger.Debugln("regexp to exclude:", excludeRegexp.String())
		}
		set := make(map[string]struct{})
		p := ""
		for i := 0; i < len(args); i++ {
			if glob {
				matched, _ := filepath.Glob(args[0])
				for _, v := range matched {
					if p == "" {
						p = v
					}
					set[v] = struct{}{}
				}
			} else {
				v := filepath.Clean(args[i])
				set[v] = struct{}{}
				if p == "" {
					p = v
				}
			}
		}
		if len(set) == 0 {
			logger.Fatalln("no files")
		}
		if output == "" {
			v, err := filepath.Abs(p)
			if err != nil {
				logger.Fatalln(err)
			}
			output = filepath.Base(v)
			if output == "" {
				logger.Fatalln("cannot get output file name")
			}
			if !strings.HasSuffix(output, ".zip") {
				output += ".zip"
			}
			logger.Infoln("auto specify output file:", output)

		} else {
			output = filepath.Clean(output)
		}
		absOutput, err := filepath.Abs(output)
		if err != nil {
			logger.Fatalln(err)
		}
		if info, err := os.Stat(output); err != nil {
			if !os.IsNotExist(err) {
				logger.Fatalln(err)
			}
		} else if info.IsDir() {
			logger.Fatalln(output, "is a directory")
		} else if overwrite {
			logger.Warnln("overwrite output file")
		} else {
			logger.Fatalln(output, "already exists, you should use --overwrite")
		}
		f, err := os.Create(output)
		if err != nil {
			logger.Fatalln(err)
		}
		var fatalErr error
		zipWriter := zip.NewWriter(f)
		start := time.Now()
		for p := range set {
			root := filepath.Base(p)
			// if p is "..", then root is ".." and archive path starts with "..", so we need get real name of ".."
			if root == ".." {
				if d, err := os.Getwd(); err != nil {
					fatalErr = err
					break
				} else {
					root = filepath.Base(filepath.Dir(d))
				}
			}
			// logger.Fatalln(root)
			fatalErr = filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				absPath, err := filepath.Abs(path)
				if err != nil {
					return err
				}
				if absPath == absOutput {
					logger.Warnln("skip", path, "because it's output file")
					return nil
				}
				relPath, err := filepath.Rel(p, path)
				if err != nil {
					return err
				}
				basename := filepath.Base(relPath)
				archivePath := filepath.Join(root, relPath)
				if runtime.GOOS == "windows" {
					archivePath = filepath.ToSlash(archivePath)
				}
				if !all && (excludeRegexp.MatchString(basename) || excludeRegexp.MatchString(archivePath)) {
					logger.Warnln("skip", path, "because it matched regexp to exclude")
					if info.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
				logger.Infoln(path, "=>", archivePath)

				header, err := zip.FileInfoHeader(info)
				if err != nil {
					return err
				}
				header.Name = archivePath
				if info.IsDir() {
					header.Name += "/" // required - strangely no mention of this in zip spec? but is in godoc...
					header.Method = zip.Store
					_, err = zipWriter.CreateHeader(header)
					return err
				}
				header.Method = zip.Deflate
				writer, err := zipWriter.CreateHeader(header)
				if err != nil {
					return err
				}
				file, err := os.Open(path)
				if err != nil {
					return err
				}
				defer file.Close()
				_, err = io.Copy(writer, file)
				return err
			})
			if fatalErr != nil {
				break
			}
		}
		zipWriter.Close()
		end := time.Now()
		f.Close()
		if fatalErr != nil {
			os.Remove(f.Name())
			logger.Fatalln(fatalErr)
		} else {
			logger.Infoln("total time:", end.Sub(start))
		}
	},
}

func init() {
	zipCmd.Flags().Bool("overwrite", false, "overwrite output file")
	zipCmd.Flags().BoolP("list", "l", false, "list files of the specified zip archive")
	zipCmd.Flags().StringP("output", "o", "", "output file")
	zipCmd.Flags().Bool("glob", false, "use glob pattern")
	zipCmd.Flags().Bool("all", false, "archive all files except output file, the regexp to exclude files will be ignored")
	zipCmd.Flags().String("exclude", `^(node_modules|__pycache__|venv|\.git)$`, "specify regexp to exclude files, first match the basename, then match the archive path")
	rootCmd.AddCommand(zipCmd)
}
