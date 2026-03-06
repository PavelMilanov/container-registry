package cmd

import (
	"os"

	"github.com/PavelMilanov/container-registry/cmd/cloud"
	"github.com/PavelMilanov/container-registry/config"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "cr",
	Short:   "local container registry",
	Version: config.VERSION,
	Run: func(cmd *cobra.Command, args []string) {
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(cloud.CloudCmd)
}
