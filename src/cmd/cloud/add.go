package cloud

import (
	"fmt"

	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "add",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := cr.AddCloud(args[0]); err != nil {
			fmt.Println(err)
			return
		}
	},
}

func init() {
	CloudCmd.AddCommand(addCmd)
}
