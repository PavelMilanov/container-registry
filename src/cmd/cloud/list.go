package cloud

import (
	"fmt"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "Показывает список пространств",
	Long:    "Команда запрашивает и выводит все доступные пространства (cloud) в реестре.",
	Example: `  cr cloud list`,
	Run: func(cmd *cobra.Command, args []string) {
		data, err := cr.GetCloudList(cmd.Context())
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
