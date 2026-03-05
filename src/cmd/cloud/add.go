package cloud

import (
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "add",
	Run: func(cmd *cobra.Command, args []string) {

	},
}

func init() {
	CloudCmd.AddCommand(addCmd)
}
