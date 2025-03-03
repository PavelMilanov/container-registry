package storage

import (
	"testing"

	"github.com/PavelMilanov/container-registry/config"
)

var env = config.NewEnv("../var/")
var s = NewStorage(env)

func TestNewStorage(t *testing.T) {
	t.Log(s)
}
