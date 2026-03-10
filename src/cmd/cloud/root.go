package cloud

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

var CloudCmd = &cobra.Command{
	Use:       "cloud",
	Short:     "cloud",
	ValidArgs: []string{"add", "del", "list", "show"},
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
