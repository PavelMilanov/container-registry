package cloud

import (
	"fmt"

	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Создаёт новое пространство",
	Long:  "Команда создаёт новое пространство (cloud), в котором будут храниться репозитории и теги образов.",
	Example: `  cr cloud add dev
  cr cloud add production`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := cr.AddCloud(cmd.Context(), args[0]); err != nil {
			fmt.Println(err)
			return
		}
	},
}

func init() {
	CloudCmd.AddCommand(addCmd)
}
