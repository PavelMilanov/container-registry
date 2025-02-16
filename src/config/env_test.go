package config

import (
	"fmt"
	"testing"
)

func TestNewEnv(t *testing.T) {
	env := Env{}
	fmt.Printf("config: %+v", env)
}
