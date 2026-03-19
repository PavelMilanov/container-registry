package settings

import (
	"fmt"
	"os"

	"github.com/PavelMilanov/container-registry/client"
	"github.com/spf13/cobra"
)

var count int

// settingsCmd represents the settings command
var setCmd = &cobra.Command{
	Use:   "set",
	Short: "Изменяет настройки хранения тегов",
	Long:  "Команда задаёт количество тегов, которые нужно сохранять в репозитории перед удалением старых.",
	Example: `  cr settings set --count 10
  cr settings set -c 5`,
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
	SettingCmd.AddCommand(setCmd)
	setCmd.Flags().IntVarP(&count, "count", "c", 0, "количество тегов для удаления")
	setCmd.MarkFlagRequired("count")
}
