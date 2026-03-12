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
		var submit string
		switch {
		case repoName != "" && tagName != "":
			if err := cr.DelImage(args[0], repoName, tagName); err != nil {
				fmt.Println(err)
				return
			}
		case repoName != "":
			fmt.Printf("Все образы будут удалены. Вы уверены? (y/n) ")
			fmt.Scan(&submit)
			if submit != "y" {
				return
			}
			if err := cr.DelRepository(args[0], repoName); err != nil {
				fmt.Println(err)
				return
			}
		default:
			fmt.Printf("Все репозитории и образы будут удалены. Вы уверены? (y/n) ")
			fmt.Scan(&submit)
			if submit != "y" {
				return
			}
			if err := cr.DelCloud(args[0]); err != nil {
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
