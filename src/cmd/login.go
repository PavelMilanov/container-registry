package cmd

import (
	"fmt"
	"os"

	"github.com/PavelMilanov/container-registry/client"
	"github.com/PavelMilanov/container-registry/config"
	"github.com/spf13/cobra"
)

var username, password string

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to the container registry",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cr := client.NewClient()
		authToken, err := cr.Login(username, password)
		if err != nil {
			fmt.Println(err)
			return
		}
		if err := os.WriteFile(config.AUTH_PATH, []byte(authToken), 0644); err != nil {
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
