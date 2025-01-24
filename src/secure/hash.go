package secure

import (
	"crypto/md5"
	"fmt"
	"io"
)

func Hashed(data string) string {
	h := md5.New()
	io.WriteString(h, data)
	hashedData := fmt.Sprintf("%x", h.Sum(nil))
	return hashedData
}
