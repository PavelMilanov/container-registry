package cmd

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/PavelMilanov/container-registry/services"
	"github.com/PavelMilanov/container-registry/storage"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

const (
	deleteOlderTagsSchedule   = "0 0 * * 0"
	garbageCollectionSchedule = "0 1 * * 0"
	uploadCleanupSchedule     = "0 2 * * *"

	uploadRetention      = 24 * time.Hour
	uploadCleanupTimeout = 5 * time.Minute
)

type cronTask struct {
	name     string
	schedule string
	timeout  time.Duration
	run      func(context.Context) (logrus.Fields, error)
}

/*
newCronScheduler создаёт планировщик фоновых заданий.

	timezone - название часового пояса для расчёта расписания.
*/
func newCronScheduler(timezone string) (*cron.Cron, error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("не удалось загрузить часовой пояс %q: %w", timezone, err)
	}

	cronLogger := cron.PrintfLogger(log.New(
		logrus.StandardLogger().WriterLevel(logrus.DebugLevel),
		"cron: ",
		log.LstdFlags,
	))

	return cron.New(
		cron.WithLocation(location),
		cron.WithLogger(cronLogger),
		cron.WithChain(
			cron.Recover(cronLogger),
			cron.SkipIfStillRunning(cronLogger),
		),
	), nil
}

/*
registerCronTasks регистрирует фоновые задания приложения.

	scheduler - планировщик cron.
	settings - repository настроек приложения.
	tagPruner - хранилище с поддержкой удаления старых тегов.
	garbageCollector - хранилище с поддержкой сборки мусора.
	uploadCleaner - хранилище с поддержкой очистки незавершённых uploads.
*/
func registerCronTasks(
	scheduler *cron.Cron,
	settings services.SettingsStore,
	tagPruner storage.TagPruner,
	garbageCollector storage.GarbageCollector,
	uploadCleaner storage.UploadCleaner,
) error {
	tasks := []cronTask{
		{
			name:     "delete_older_tags",
			schedule: deleteOlderTagsSchedule,
			run: func(ctx context.Context) (logrus.Fields, error) {
				return nil, services.DeleteOlderTags(
					ctx,
					settings,
					tagPruner,
				)
			},
		},
		{
			name:     "garbage_collection",
			schedule: garbageCollectionSchedule,
			run: func(context.Context) (logrus.Fields, error) {
				return nil, services.GarbageCollection(garbageCollector)
			},
		},
		{
			name:     "upload_cleanup",
			schedule: uploadCleanupSchedule,
			timeout:  uploadCleanupTimeout,
			run: func(ctx context.Context) (logrus.Fields, error) {
				deleted, err := uploadCleaner.CleanupUploads(
					ctx,
					uploadRetention,
				)
				return logrus.Fields{
					"deleted":   deleted,
					"retention": uploadRetention,
				}, err
			},
		},
	}

	for _, task := range tasks {
		if _, err := scheduler.AddFunc(task.schedule, task.handler()); err != nil {
			return fmt.Errorf("не удалось зарегистрировать cron-задачу %s: %w", task.name, err)
		}
	}

	return nil
}

/*
handler создаёт функцию запуска cron-задачи с единым логированием.

Для задания с timeout создаёт context с ограниченным временем выполнения.
*/
func (task cronTask) handler() func() {
	return func() {
		ctx := context.Background()
		cancel := func() {}
		if task.timeout > 0 {
			ctx, cancel = context.WithTimeout(ctx, task.timeout)
		}
		defer cancel()

		logger := logrus.WithFields(logrus.Fields{
			"task":     task.name,
			"schedule": task.schedule,
			"timeout":  task.timeout,
		})
		logger.Info("Запуск cron-задачи")

		fields, err := task.run(ctx)
		if fields != nil {
			logger = logger.WithFields(fields)
		}
		if err != nil {
			logger.WithError(err).Error("Cron-задача завершилась с ошибкой")
			return
		}

		logger.Info("Cron-задача завершена")
	}
}
