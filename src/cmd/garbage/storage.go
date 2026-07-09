package garbage

import (
	"fmt"
	"os"

	"github.com/PavelMilanov/container-registry/client"
	"github.com/spf13/cobra"
)

// garbadgeCmd represents the garbage collection command
var storageCmd = &cobra.Command{
	Use:     "storage",
	Short:   "Запускает сборку мусора слоёв",
	Long:    "Команда запускает серверную процедуру garbage collection для удаления неиспользуемых слоёв.",
	Example: `  cr garbage storage`,
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cr := client.NewClient()
		if err := cr.GarbageCollection(cmd.Context()); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func init() {
	GarbageCmd.AddCommand(storageCmd)
}
