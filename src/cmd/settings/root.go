package settings

import (
	"fmt"

	"github.com/PavelMilanov/container-registry/client"
	"github.com/PavelMilanov/container-registry/config"
	"github.com/spf13/cobra"
)

var (
	cr  *client.Client
	env *config.Env
)

var SettingCmd = &cobra.Command{
	Use:       "settings",
	Short:     "Управление настройками реестра",
	Long:      "Группа команд для просмотра и изменения параметров реестра, связанных с политикой хранения тегов.",
	Example:   "  cr settings show\n  cr settings set --count 10",
	ValidArgs: []string{"set", "show"},
	Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
	},
}

func init() {
	cgf, err := config.NewEnv(config.CONFIG_PATH, "config")
	if err != nil {
		fmt.Println(err)
		return
	}
	cr = client.NewClient()
	env = cgf
}
