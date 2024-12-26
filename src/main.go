package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/PavelMilanov/container-registry/handlers"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetLevel(logrus.TraceLevel)
	logrus.SetReportCaller(true)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006/01/02 15:04:00",
	})

	logrus.Debug("Версия сборки: ", config.VERSION)

	handler := handlers.NewHandler()
	srv := new(Server)
	go func() {
		if err := srv.Run(handler.InitRouters()); err != nil {
			logrus.Warn(err)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logrus.Infof("Сигнал остановки сервера через %d секунды\n", config.DURATION)
	if err := srv.Shutdown(time.Duration(config.DURATION)); err != nil {
		logrus.WithError(err).Error("ошибка при остановке сервера")
	}
}
