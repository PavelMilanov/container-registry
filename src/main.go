/*
Copyright 2025 Pavel Milanov
Licensed under the Apache License, Version 2.0 (see LICENSE file)
*/
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
	"github.com/PavelMilanov/container-registry/services"
	"github.com/PavelMilanov/container-registry/storage"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetLevel(logrus.TraceLevel)
	logrus.SetReportCaller(true)
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
	sqlite, err := db.NewDatabase(sqliteFIle)
	if err != nil {
		logrus.Fatal(err)
	}
	defer db.CloseDatabase(sqlite.Sql)

	c.AddFunc("0 1 * * 0", func() {
		logrus.Info("Запуск задания Garbage Collection")
		go store.GarbageCollection()
	}) // каждое воскресенье в 01:00
	c.AddFunc("0 0 * * 0", func() {
		logrus.Info("Запуск задания удаления старых образов")
		go services.DeleteOlderImages(sqlite.Sql, store)
	}) // каждое воскресенье в 00:00

	handler := handlers.NewHandler(store, &sqlite, env)
	srv := new(Server)
	go func() {
		if err := srv.Run(handler.InitRouters()); err != nil {
			logrus.Warn(err)
		}
	}()

	c.Start()
	defer c.Stop()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logrus.Infof("Сигнал остановки сервера через %d секунды\n", config.DURATION)
	if err := srv.Shutdown(time.Duration(config.DURATION)); err != nil {
		logrus.WithError(err).Error("ошибка при остановке сервера")
	}
}
