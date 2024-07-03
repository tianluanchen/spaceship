package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version = "0.0.0"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of spaceship",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("spaceship version", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
