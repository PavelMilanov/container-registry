/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/PavelMilanov/container-registry/client"
	"github.com/spf13/cobra"
)

// healtcheckCmd represents the healtcheck command
var healthcheckCmd = &cobra.Command{
	Use:   "healthcheck",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		cr := client.NewClient()
		if err := cr.HealthCheck(); err != nil {
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(healthcheckCmd)
}
