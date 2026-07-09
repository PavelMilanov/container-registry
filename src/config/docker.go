package config

var MANIFEST_TYPE = map[string]string{
	"manifest": "application/vnd.oci.image.manifest.v1+json",
	"index":    "application/vnd.oci.image.index.v1+json",
}

// Абстракция application/vnd.oci.image.manifest.v1+json
type Manifest struct {
	MediaType string `json:"mediaType"`
	Config    struct {
		Digest string `json:"digest"`
		Size   int64  `json:"size"`
	} `json:"config"`
	Layers []struct {
		Digest string `json:"digest"`
		Size   int64  `json:"size"`
	} `json:"layers"`
}

// Абстракция application/vnd.oci.image.index.v1+json
type Index struct {
	MediaType string `json:"mediaType"`
	Manifests []struct {
		Digest   string `json:"digest"`
		Size     int64  `json:"size"`
		Platform struct {
			Architecture string `json:"architecture"`
			OS           string `json:"os"`
		} `json:"platform"`
	} `json:"manifests"`
}

/*
Blob абстракция

	application/vnd.docker.image.rootfs.diff.tar.gzip
*/
type Blob struct {
	Size   int64
	Digest string
	Path   string
}

/*
Meta абстракция для метаданных манифеста
*/
type Meta struct {
	Repository string
	Image      string
	Tag        string
	MediaType  string
	Digest     string
	Platform   string
	Size       int64
}
