/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/PavelMilanov/container-registry/db"
	"github.com/PavelMilanov/container-registry/handlers"
	"github.com/PavelMilanov/container-registry/services"
	"github.com/PavelMilanov/container-registry/storage"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006/01/02 15:04:00",
		})
		env, err := config.NewEnv(config.CONFIG_PATH, "config")
		if err != nil {
			logrus.Fatal(err)
		}
		store, err := storage.NewStorage(env)
		if err != nil {
			logrus.Fatal(err)
		}
		location, _ := time.LoadLocation(os.Getenv("TZ"))
		c := cron.New(
			cron.WithLocation(location),
		)

		sqliteFIle := fmt.Sprintf("%s/registry.db", config.DATA_PATH)
		sqlite, err := db.NewDatabase(sqliteFIle, env)
		if err != nil {
			logrus.Fatal(err)
		}
		defer db.CloseDatabase(sqlite.Sql)

		_, err = c.AddFunc("0 1 * * 0", func() {
			logrus.Debug("Запуск задания Garbage Collection")
			go store.GarbageCollection()
		}) // каждое воскресенье в 01:00
		if err != nil {
			logrus.Error(err)
		}
		_, err = c.AddFunc("0 0 * * 0", func() {
			logrus.Debug("Запуск задания удаления старых образов")
			go services.DeleteOlderImages(sqlite.Sql, store)
		}) // каждое воскресенье в 00:00
		if err != nil {
			logrus.Error(err)
		}
		c.Start()
		logrus.Debugf("Запущено %d заданий планировщика", len(c.Entries()))

		handler := handlers.NewHandler(store, &sqlite, env)
		srv := new(config.Server)
		go func() {
			if err := srv.Run(handler.InitRouters()); err != nil {
				logrus.Warn(err)
			}
		}()

		defer c.Stop()

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		logrus.Infof("Сигнал остановки сервера через %d секунды\n", config.DURATION)
		if err := srv.Shutdown(time.Duration(config.DURATION)); err != nil {
			logrus.WithError(err).Error("ошибка при остановке сервера")
		}
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
