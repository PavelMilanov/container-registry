package cloud

import (
	"fmt"

	"github.com/PavelMilanov/container-registry/client"
	"github.com/PavelMilanov/container-registry/config"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list",
	Run: func(cmd *cobra.Command, args []string) {
		env, err := config.NewEnv(config.CONFIG_PATH, "config")
		if err != nil {
			fmt.Println(err)
			return
		}
		authToken, err := client.Login(env.User.Login, env.User.Password)
		if err != nil {
			fmt.Println(err)
			return
		}
		data, err := client.GetCloudList(authToken)
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
