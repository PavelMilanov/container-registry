package storage

import (
	"testing"
)

func TestNewS3storage(t *testing.T) {
	s3 := NewS3storage("192.168.12.27:9001", "jlSDwyyoeI71kbnRC1wK", "sBqscm1bmreddLINb5aQqSU2gq6qbmpeqVZ9OxVK")
	t.Log(s3)
}
