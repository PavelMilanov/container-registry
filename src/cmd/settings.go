package cmd

import (
	"fmt"
	"os"

	"github.com/PavelMilanov/container-registry/client"
	"github.com/spf13/cobra"
)

var count int

// settingsCmd represents the settings command
var settingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cr := client.NewClient()
		toStr := fmt.Sprintf("%d", count)
		if err := cr.SetGarbageTagCount(toStr); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(settingsCmd)
	settingsCmd.Flags().IntVarP(&count, "count", "c", 0, "количество тегов для удаления")
	settingsCmd.MarkFlagRequired("count")
}
