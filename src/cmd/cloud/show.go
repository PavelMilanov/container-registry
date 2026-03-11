package cloud

import (
	"fmt"

	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "show",
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
			images, err := cr.GetImagesList(authToken, args[0], item)
			if err != nil {
				fmt.Println(err)
				continue
			}
			for _, img := range images {
				fmt.Printf(" - %s/%s:%s\n", args[0], item, img)
			}
		}
	},
}

func init() {
	CloudCmd.AddCommand(showCmd)
}
