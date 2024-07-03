package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "install to GOPATH BIN",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		overwrite, _ := cmd.Flags().GetBool("overwrite")
		exePath, err := os.Executable()
		if err != nil {
			logger.Fatalln(err)
		}
		logger.Infoln("File:", exePath)
		gopath := os.Getenv("GOPATH")
		gopathList := strings.Split(gopath, string(os.PathListSeparator))
		if gopathList[0] == "" {
			logger.Fatalln("GOPATH is not set")
		}
		name := "spaceship"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		binPath := filepath.Join(gopathList[0], "bin", name)
		logger.Infoln("Install to", binPath)
		if binPath == exePath {
			logger.Warnln("Already install!")
			return
		}
		if info, err := os.Stat(binPath); !os.IsNotExist(err) {
			if err != nil {
				logger.Fatalln(err)
			}
			if info.IsDir() {
				logger.Fatalln(binPath, "is a directory!")
			}
			logger.Warnln("Already exists file!")
			if !overwrite {
				var answer string
				fmt.Print("Continue to overwrite? (y/n): ")
				fmt.Scanln(&answer)
				if strings.Trim(strings.ToLower(answer), "\r\n\t ") != "y" {
					logger.Warnln("Stop install")
					return
				}
			}
			logger.Warnln("Overwrite", binPath)
		}
		src, err := os.Open(exePath)
		if err != nil {
			logger.Fatalln(err)
		}
		defer src.Close()
		dst, err := os.Create(binPath)
		if err != nil {
			logger.Fatalln(err)
		}
		defer dst.Close()
		_, err = io.Copy(dst, src)
		if err != nil {
			logger.Fatalln(err)
		}
		if err := os.Chmod(binPath, 0755); err != nil {
			logger.Fatalln(err)
		} else {
			logger.Infoln("Install success, run", name, "to start")
		}
	},
}

func init() {
	installCmd.Flags().Bool("overwrite", false, "overwrite if exists")
	rootCmd.AddCommand(installCmd)
}
