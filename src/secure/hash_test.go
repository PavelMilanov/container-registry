package secure

import "testing"

func TestHashed(t *testing.T) {
	data1 := "hello world"
	result := Hashed(data1)
	err := ValidateHash(data1, []byte(result))
	if err != nil {
		t.Error(err.Error())
	}
}
