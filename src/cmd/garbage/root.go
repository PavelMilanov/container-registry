package garbage

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

var GarbageCmd = &cobra.Command{
	Use:       "garbage",
	Short:     "Ручной запуск операций очистки",
	Long:      "Группа команд для ручной очистки хранилища: сборка мусора слоёв и удаление старых тегов.",
	Example:   "  cr garbage storage\n  cr garbage tags",
	ValidArgs: []string{"storage", "tags"},
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
