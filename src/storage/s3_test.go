package storage

import "testing"

func TestNewS3(t *testing.T) {
	testS3 := newS3("192.168.12.27:9000", "J3BwPUGqDPWJTeZd5Dcv", "hHgZgIyuuHKuvtmOcyY4NV2cotkgGft93VZnHkPW", false)
	t.Log(testS3)
}
