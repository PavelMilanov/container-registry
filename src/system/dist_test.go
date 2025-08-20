package system

import (
	"testing"
)

func TestDiskUsage(t *testing.T) {
	stat, err := DiskUsage()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Total: %s, Free: %s", HumanizeSize(stat.Total), HumanizeSize(stat.Free))
}
