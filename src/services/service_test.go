package services

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/PavelMilanov/container-registry/config"
)

func TestSaveManifestToDB(t *testing.T) {
	link := "../var/manifests/local/registry/sha256:91ee867b7ee1c48b01a932d593450babd01eb3fd43594d2aa6855c07c0a5b3fc"
	func(link string) {
		var manifest config.Manifest
		body, err := os.ReadFile(link)
		if err != nil {
			t.Log(err)
		}
		if err := json.Unmarshal(body, &manifest); err != nil {
			t.Log(err)
		}
		platforms := []string{}
		sizes := []int64{}
		for _, item := range manifest.Manifests {
			if item.Platform.Architecture != "unknown" {
				platforms = append(platforms, item.Platform.OS+"/"+item.Platform.Architecture)
				path := "../var/manifests/local/registry/"
				body, err := os.ReadFile(path + item.Digest)
				if err != nil {
					t.Log(err)
				}
				var m2 config.Manifest
				if err := json.Unmarshal(body, &m2); err != nil {
					t.Log(err)
				}
				var sum int64
				for _, descriptor := range m2.Layers {
					sum += descriptor.Size
				}
				sizes = append(sizes, sum)
			}
		}
		fmt.Println(platforms)
		fmt.Println(sizes)
	}(link)
}
