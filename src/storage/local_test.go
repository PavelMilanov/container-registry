package storage

import (
	"testing"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/PavelMilanov/container-registry/system"
)

func TestDiskUsage(t *testing.T) {
	env, _ := config.NewEnv("../", "test.config")
	store, err := NewStorage(env)
	if err != nil {
		t.Fatal(err)
	}
	stat, err := store.DiskUsage()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Total: %s, Free: %s, Usage: %d%%", system.HumanizeSize(stat.Total), system.HumanizeSize(stat.Used), int(stat.UsedToPercent))
}
