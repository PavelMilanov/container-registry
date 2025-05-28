package config

type Descriptor struct {
	MediaType string `json:"mediaType"`
	Size      int64  `json:"size"`
	Digest    string `json:"digest"`
}

type Platform struct {
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
}

/*
ManifestOCI абстракция

	application/vnd.oci.image.manifest.v1+json
*/
type ManifestOCI struct {
	MediaType string   `json:"mediaType"`
	Digest    string   `json:"Digest"`
	Size      int64    `json:"Size"`
	Platform  Platform `json:"platform"`
}

/*
Manifest абстракция

	application/vnd.docker.distribution.manifest.v2+json
*/
type Manifest struct {
	SchemaVersion int           `json:"schemaVersion"`
	MediaType     string        `json:"mediaType"`
	Config        Descriptor    `json:"config"`
	Layers        []Descriptor  `json:"layers"`
	Manifests     []ManifestOCI `json:"manifests"`
}

type Blob struct {
	Size   int64
	Digest string
}
