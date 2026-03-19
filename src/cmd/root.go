package cmd

import (
	"os"

	"github.com/PavelMilanov/container-registry/cmd/cloud"
	"github.com/PavelMilanov/container-registry/cmd/garbage"
	"github.com/PavelMilanov/container-registry/cmd/settings"
	"github.com/PavelMilanov/container-registry/config"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cr",
	Short: "CLI для управления локальным container registry",
	Long: `cr — командная утилита для работы с локальным контейнерным реестром.
Позволяет запускать сервер и/или выполнятьо операции клиента`,
	Version:   config.VERSION,
	ValidArgs: []string{"healthcheck", "settings", "login", "cloud", "serve", "garbage"},
	Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(cloud.CloudCmd)
	rootCmd.AddCommand(settings.SettingCmd)
	rootCmd.AddCommand(garbage.GarbageCmd)
}
