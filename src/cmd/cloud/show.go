package cloud

import (
	"fmt"

	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:     "show",
	Short:   "Показывает репозитории и теги в пространстве",
	Long:    "Команда выводит содержимое указанного пространства: репозитории и их теги образов.",
	Example: `  cr cloud show dev`,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		data, err := cr.GetRepositoriesList(cmd.Context(), args[0])
		if err != nil {
			fmt.Println(err)
			return
		}
		for _, item := range data {
			images, err := cr.GetImagesList(cmd.Context(), args[0], item)
			if err != nil {
				fmt.Println(err)
				continue
			}
			for _, img := range images {
				fmt.Printf(" - %s/%s:%s\n", args[0], item, img)
			}
		}
	},
}

func init() {
	CloudCmd.AddCommand(showCmd)
}
