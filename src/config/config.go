// Package config реализует управление конфигурацией приложения.
package config

import "path/filepath"

var (
	DURATION = 5
	VERSION  string

	DATA_PATH     = "var"
	CONFIG_PATH   = "conf.d"
	STORAGE_PATH  = filepath.Join(DATA_PATH)
	MANIFEST_PATH = filepath.Join(DATA_PATH, "manifests")
	BLOBS_PATH    = filepath.Join(DATA_PATH, "blobs")
	TMP_PATH      = filepath.Join(DATA_PATH, "tmp")

	BACKET_NAME              = "registry"
	DEFAULT_TAG_EXPIRED_DAYS = 0
)
