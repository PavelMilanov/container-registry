//go:build dev

package main

import "github.com/sirupsen/logrus"

func setLogger() {
	logrus.SetReportCaller(true)
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006/01/02 15:04:00",
	})
}
