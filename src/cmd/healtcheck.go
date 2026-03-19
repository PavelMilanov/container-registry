/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/PavelMilanov/container-registry/client"
	"github.com/spf13/cobra"
)

// healtcheckCmd represents the healtcheck command
var healthcheckCmd = &cobra.Command{
	Use:   "healthcheck",
	Short: "Проверяет доступность API реестра",
	Long: `Команда отправляет healthcheck-запрос на сервер реестра.
Если сервис недоступен или возвращает ошибку, команда завершается с ненулевым кодом.`,
	Example: `  cr healthcheck`,
	Run: func(cmd *cobra.Command, args []string) {
		cr := client.NewClient()
		if err := cr.HealthCheck(); err != nil {
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(healthcheckCmd)
}
