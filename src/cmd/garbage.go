package cmd

import (
	"os"

	"github.com/PavelMilanov/container-registry/client"
	"github.com/spf13/cobra"
)

// garbadgeCmd represents the garbage collection command
var garbadgeCmd = &cobra.Command{
	Use:   "garbage",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cr := client.NewClient()
		if err := cr.GarbageCollection(); err != nil {
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(garbadgeCmd)
}
