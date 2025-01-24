package config

var (
	DURATION   = 3
	VERSION    string
	JWT_SECRET = []byte("super-secret-key")

	STORAGE_PATH  = "data"
	MANIFEST_PATH = "manifests"
	BLOBS_PATH    = "blobs"

	DATA_PATH = "var"

	WEB_API_URL string
)
