package settings

import (
	"fmt"
	"os"

	"github.com/PavelMilanov/container-registry/client"
	"github.com/spf13/cobra"
)

// settingsCmd represents the settings command
var getCmd = &cobra.Command{
	Use:     "show",
	Short:   "Показывает текущие настройки хранения тегов",
	Long:    "Команда запрашивает и выводит текущее значение параметра хранения тегов из настроек реестра.",
	Example: `  cr settings show`,
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cr := client.NewClient()
		if err := cr.GetGarbageTagCount(cmd.Context()); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func init() {
	SettingCmd.AddCommand(getCmd)
}
