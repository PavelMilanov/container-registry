package cloud

import (
	"github.com/spf13/cobra"
)

var delCmd = &cobra.Command{
	Use:   "del",
	Short: "del",
	Run: func(cmd *cobra.Command, args []string) {

	},
}

func init() {
	CloudCmd.AddCommand(delCmd)
}
