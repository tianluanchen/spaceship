package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tianluanchen/spaceship/ship"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of spaceship",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("spaceship version", ship.Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
