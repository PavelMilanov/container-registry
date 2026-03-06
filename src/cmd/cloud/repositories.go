package cloud

import (
	"fmt"

	"github.com/spf13/cobra"
)

var RepositoriesCmd = &cobra.Command{
	Use:   "repositories",
	Short: "repositories",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		authToken, err := cr.Login(env.User.Login, env.User.Password)
		if err != nil {
			fmt.Println(err)
			return
		}
		data, err := cr.GetRepositoriesList(authToken, args[0])
		if err != nil {
			fmt.Println(err)
			return
		}
		for _, item := range data {
			fmt.Printf(" - %s/%s\n", args[0], item)
		}
	},
}

func init() {
	CloudCmd.AddCommand(RepositoriesCmd)
}
