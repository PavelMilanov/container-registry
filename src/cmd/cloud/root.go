package cloud

import (
	"github.com/spf13/cobra"
)

var CloudCmd = &cobra.Command{
	Use:       "cloud",
	Short:     "cloud",
	ValidArgs: []string{"add", "del", "list"},
	Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {

	},
}

func init() {
}
