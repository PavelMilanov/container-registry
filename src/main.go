package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
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

func init() {
	storage := storage.NewStorage()
	blobPath := filepath.Join(storage.BlobPath)
	manifestPath := filepath.Join(storage.ManifestPath)
	os.MkdirAll(blobPath, 0755)
	os.MkdirAll(manifestPath, 0755)
	os.Mkdir(config.DATA_PATH, 0755)

}

func main() {
	logrus.SetLevel(logrus.TraceLevel)
	logrus.SetReportCaller(true)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006/01/02 15:04:00",
	})
	c := cron.New(
		cron.WithSeconds(),
		cron.WithLocation(time.Local),
	)
	defer c.Stop()

	_, err := c.AddFunc("0 0 0 * * 0", system.GarbageCollection) // каждую неделю
	if err != nil {
		logrus.Error("Ошибка фоновой задачи:", err)
	}
	storage := storage.NewStorage()

	sqliteFIle := fmt.Sprintf("%s/registry.db", config.DATA_PATH)
	sqlite := db.NewDatabase(sqliteFIle)
	defer db.CloseDatabase(sqlite.Sql)

	handler := handlers.NewHandler(storage, &sqlite)
	srv := new(Server)
	go func() {
		if err := srv.Run(handler.InitRouters()); err != nil {
			logrus.Warn(err)
		}
	}()

	c.Start()
	logrus.Info("Планировщик запущен")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logrus.Infof("Сигнал остановки сервера через %d секунды\n", config.DURATION)
	if err := srv.Shutdown(time.Duration(config.DURATION)); err != nil {
		logrus.WithError(err).Error("ошибка при остановке сервера")
	}
}
