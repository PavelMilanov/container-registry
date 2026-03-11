package cloud

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	repoName string
	tagName  string
)

var delCmd = &cobra.Command{
	Use:   "del",
	Short: "del",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		authToken, err := cr.Login(env.User.Login, env.User.Password)
		if err != nil {
			fmt.Println(err)
			return
		}
		switch {
		case repoName != "" && tagName != "":
			if err := cr.DelImage(authToken, args[0], repoName, tagName); err != nil {
				fmt.Println(err)
				return
			}
		case repoName != "":
			// if err := cr.DelRepo(authToken, args[0], repoName); err != nil {
			// 	fmt.Println(err)
			// 	return
			// }
		default:
			if err := cr.DelCloud(authToken, args[0]); err != nil {
				fmt.Println(err)
				return
			}
		}
	},
}

func init() {
	CloudCmd.AddCommand(delCmd)
	delCmd.Flags().StringVarP(&repoName, "repo", "r", "", "имя репозитория")
	delCmd.Flags().StringVarP(&tagName, "tag", "t", "", "имя тега")
}
