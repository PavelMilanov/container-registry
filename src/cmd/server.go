/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"
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
	Use:     "serve",
	Short:   "Запускает HTTP-сервер реестра",
	Long:    `Команда запускает API-сервер container registry.`,
	Example: `  cr serve`,
	Args:    cobra.NoArgs,
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
		cronLogger := cron.VerbosePrintfLogger(log.New(
			logrus.StandardLogger().WriterLevel(logrus.DebugLevel),
			"cron: ",
			log.LstdFlags,
		))
		c := cron.New(
			cron.WithLocation(location),
			cron.WithLogger(cronLogger),
			cron.WithChain(
				cron.Recover(cronLogger),
				cron.SkipIfStillRunning(cronLogger),
			),
		)

		sqliteFIle := fmt.Sprintf("%s/registry.db", config.DATA_PATH)
		sqlite, err := db.NewDatabase(sqliteFIle, env)
		if err != nil {
			logrus.Fatal(err)
		}
		defer db.CloseDatabase(sqlite.Sql)

		_, err = c.AddFunc("0 0 * * 0", func() {
			logrus.WithField("Garbage Collection", "start").Info("Запуск задания по удалению старых тегов")
			if err := services.DeleteOlderTags(sqlite.Sql, store); err != nil {
				logrus.WithError(err).Error("Ошибка при удалении старых тегов")
			}
			logrus.WithField("Garbage Collection", "end").Info("Завершение задания по удалению старых тегов")
		}) // каждое воскресенье в 00:00
		if err != nil {
			logrus.WithError(err).Error("Не удалось добавить cron-задачу удаления старых тегов")
		}
		_, err = c.AddFunc("0 1 * * 0", func() {
			logrus.WithField("Garbage Collection", "start").Info("Запуск задания по сборке мусора")
			if err := services.GarbageCollection(store); err != nil {
				logrus.WithError(err).Error("Ошибка при сборке мусора")
			}
			logrus.WithField("Garbage Collection", "end").Info("Завершение задания по сборке мусора")
		}) // каждое воскресенье в 01:00
		if err != nil {
			logrus.WithError(err).Error("Не удалось добавить cron-задачу сборки мусора")
		}
		c.Start()
		logrus.WithField("task", "Garbage Collection").Infof("Запущено %d заданий планировщика", len(c.Entries()))

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
