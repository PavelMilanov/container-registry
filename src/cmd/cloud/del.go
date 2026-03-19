package cloud

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	repoName string
	tagName  string
)

var delCmd = &cobra.Command{
	Use:   "del",
	Short: "Удаляет пространство, репозиторий или тег",
	Long: `Команда удаляет данные в зависимости от переданных флагов:
- без флагов: удаляет целое пространство;
- с --repo: удаляет репозиторий;
- с --repo и --tag: удаляет конкретный тег образа.`,
	Example: `  cr cloud del dev
  cr cloud del dev -r api
  cr cloud del dev -r api -t latest`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var submit string
		switch {
		case repoName != "" && tagName != "":
			if err := cr.DelImage(args[0], repoName, tagName); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		case repoName != "":
			fmt.Printf("Все образы будут удалены. Вы уверены? (y/n) ")
			fmt.Scan(&submit)
			if submit != "y" {
				os.Exit(0)
			}
			if err := cr.DelRepository(args[0], repoName); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		default:
			fmt.Printf("Все репозитории и образы будут удалены. Вы уверены? (y/n) ")
			fmt.Scan(&submit)
			if submit != "y" {
				os.Exit(0)
			}
			if err := cr.DelCloud(args[0]); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		}
	},
}

func init() {
	CloudCmd.AddCommand(delCmd)
	delCmd.Flags().StringVarP(&repoName, "repo", "r", "", "имя репозитория")
	delCmd.Flags().StringVarP(&tagName, "tag", "t", "", "имя тега")
}
