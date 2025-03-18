// Package config реализует управление конфигурацией приложения.
package config

import "path/filepath"

var (
	DURATION = 5
	VERSION  string

	DATA_PATH     = "var"
	STORAGE_PATH  = filepath.Join(DATA_PATH, "data")
	MANIFEST_PATH = filepath.Join(DATA_PATH, "data", "manifests")
	BLOBS_PATH    = filepath.Join(DATA_PATH, "data", "blobs")
	TMP_PATH      = filepath.Join(DATA_PATH, "tmp")

	BACKET_NAME              = "registry"
	DEFAULT_TAG_EXPIRED_DAYS = 0
)
