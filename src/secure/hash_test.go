package secure

import "testing"

func TestHashed(t *testing.T) {
	data := []string{"a", "b", "c", "d", "a"}
	for _, item := range data {
		result := Hashed(item)
		t.Log(result)
	}
}
