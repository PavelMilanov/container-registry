package cloud

import (
	"fmt"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list",
	Run: func(cmd *cobra.Command, args []string) {
		authToken, err := cr.Login(env.User.Login, env.User.Password)
		if err != nil {
			fmt.Println(err)
			return
		}
		data, err := cr.GetCloudList(authToken)
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
