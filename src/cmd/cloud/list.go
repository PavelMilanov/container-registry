package cloud

import (
	"fmt"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list",
	Run: func(cmd *cobra.Command, args []string) {
		data, err := cr.GetCloudList()
		if err != nil {
			fmt.Println(err)
			return
		}
		for _, item := range data {
			fmt.Println(" - " + item)
		}
	},
}

func init() {
	CloudCmd.AddCommand(listCmd)
}
