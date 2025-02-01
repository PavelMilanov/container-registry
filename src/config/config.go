package config

import "os"

var (
	DURATION   = 3
	VERSION    string
	JWT_SECRET = []byte(os.Getenv("JWT_KEY"))

	STORAGE_PATH  = "data"
	MANIFEST_PATH = "manifests"
	BLOBS_PATH    = "blobs"

	DATA_PATH = "var"

	URL = os.Getenv("URL")
)
