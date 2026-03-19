package garbage

import (
	"fmt"
	"os"

	"github.com/PavelMilanov/container-registry/client"
	"github.com/spf13/cobra"
)

// garbadgeCmd represents the garbage collection command
var tagsCmd = &cobra.Command{
	Use:     "tags",
	Short:   "Удаляет старые теги",
	Long:    "Команда запускает серверную очистку старых тегов в соответствии с параметром хранения тегов.",
	Example: `  cr garbage tags`,
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cr := client.NewClient()
		if err := cr.GarbageTags(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func init() {
	GarbageCmd.AddCommand(tagsCmd)
}
