package cmd

import (
	"fmt"

	"github.com/PavelMilanov/container-registry/client"
	"github.com/spf13/cobra"
)

var username, password string

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:     "login",
	Short:   "Авторизует пользователя в реестре",
	Long:    `Команда выполняет вход по логину и паролю через API реестра.`,
	Example: `  cr login -u admin -p secret`,
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cr := client.NewClient()
		authToken, err := cr.Login(cmd.Context(), username, password)
		if err != nil {
			fmt.Println(err)
			return
		}
		if err := cr.SetToken(authToken); err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("Добро пожаловать, %s!\n", username)
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
	loginCmd.Flags().StringVarP(&username, "username", "u", "", "имя пользователя")
	loginCmd.Flags().StringVarP(&password, "password", "p", "", "пароль")
	loginCmd.MarkFlagRequired("username")
	loginCmd.MarkFlagRequired("password")
}
