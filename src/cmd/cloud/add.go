package cloud

import (
	"fmt"

	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "add",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		authToken, err := cr.Login(env.User.Login, env.User.Password)
		if err != nil {
			fmt.Println(err)
			return
		}
		if err := cr.AddCloud(authToken, args[0]); err != nil {
			fmt.Println(err)
			return
		}
	},
}

func init() {
	CloudCmd.AddCommand(addCmd)
}
