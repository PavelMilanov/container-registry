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
	registryauth "github.com/PavelMilanov/container-registry/internal/auth"
	"github.com/PavelMilanov/container-registry/services"
	"github.com/PavelMilanov/container-registry/storage"
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
		backend, err := storage.NewStorage(env)
		if err != nil {
			logrus.Fatal(err)
		}
		if backend.Uploads == nil {
			logrus.Fatalf(
				"storage %q не поддерживает загрузку blob",
				env.Storage.Type,
			)
		}
		if backend.UploadCleaner == nil {
			logrus.Fatalf(
				"storage %q не поддерживает очистку незавершённых uploads",
				env.Storage.Type,
			)
		}

		sqliteFIle := fmt.Sprintf("%s/registry.db", config.DATA_PATH)
		sqlite, err := db.NewDatabase(sqliteFIle)
		if err != nil {
			logrus.Fatal(err)
		}
		defer db.CloseDatabase(sqlite.Sql)

		passwords := registryauth.NewPasswordHasher()
		tokens, err := registryauth.NewTokenManager(registryauth.TokenConfig{
			Secret:   []byte(env.Server.Jwt),
			Issuer:   env.Server.Issuer,
			Audience: env.Server.Service,
			TTL:      env.Server.TokenTTL,
		})
		if err != nil {
			logrus.Fatal(err)
		}
		authService := services.NewAuthService(
			db.NewUserRepository(sqlite.Sql),
			passwords,
			tokens,
		)
		if err := authService.EnsureUser(
			cmd.Context(),
			env.User.Login,
			env.User.Password,
		); err != nil {
			logrus.Fatal(err)
		}

		scheduler, err := newCronScheduler(os.Getenv("TZ"))
		if err != nil {
			logrus.Fatal(err)
		}
		if err := registerCronTasks(
			scheduler,
			&sqlite,
			backend.TagPruner,
			backend.GarbageCollector,
			backend.UploadCleaner,
		); err != nil {
			logrus.Fatal(err)
		}

		scheduler.Start()
		logrus.WithField("tasks", len(scheduler.Entries())).
			Info("Задачи планировщика запущены")

		handler := handlers.NewHandler(
			handlers.StorageDependencies{
				Blobs:            backend.Blobs,
				Uploads:          backend.Uploads,
				Manifests:        backend.Manifests,
				Namespaces:       backend.Namespaces,
				Clouds:           backend.Clouds,
				Repositories:     backend.Repositories,
				Tags:             backend.Tags,
				GarbageCollector: backend.GarbageCollector,
				TagPruner:        backend.TagPruner,
			},
			authService,
			&sqlite,
			env,
		)
		srv := new(config.Server)
		go func() {
			if err := srv.Run(handler.InitRouters()); err != nil {
				logrus.Warn(err)
			}
		}()

		defer func() {
			cronCtx := scheduler.Stop()
			<-cronCtx.Done()
		}()

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
