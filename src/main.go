package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/PavelMilanov/container-registry/db"
	"github.com/PavelMilanov/container-registry/handlers"
	"github.com/PavelMilanov/container-registry/storage"
	"github.com/PavelMilanov/container-registry/system"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

func main() {
	env := config.NewEnv()

	storage := storage.NewStorage(env)

	logrus.SetLevel(logrus.TraceLevel)
	logrus.SetReportCaller(true)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006/01/02 15:04:00",
	})
	location, _ := time.LoadLocation(os.Getenv("TZ"))
	c := cron.New(
		cron.WithLocation(location),
	)

	c.AddFunc("0 0 * * 0", func() {
		system.GarbageCollection(storage)
	}) // каждую неделю

	sqliteFIle := fmt.Sprintf("%s/registry.db", config.DATA_PATH)
	sqlite := db.NewDatabase(sqliteFIle)
	defer db.CloseDatabase(sqlite.Sql)

	handler := handlers.NewHandler(storage, &sqlite, env)
	srv := new(Server)
	go func() {
		if err := srv.Run(handler.InitRouters()); err != nil {
			logrus.Warn(err)
		}
	}()

	c.Start()
	defer c.Stop()

	logrus.Info("Планировщик запущен")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logrus.Infof("Сигнал остановки сервера через %d секунды\n", config.DURATION)
	if err := srv.Shutdown(time.Duration(config.DURATION)); err != nil {
		logrus.WithError(err).Error("ошибка при остановке сервера")
	}
}
