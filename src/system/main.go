package system

import "fmt"

func ConvertSize(size int64) string {
	MB := int64(1000000)
	GB := int64(1000000000)
	if size < MB || size > MB && size < GB {
		flt := float64(size) / 1000000
		sizeToString := fmt.Sprintf("%.1f МБ", flt)
		return sizeToString
	}
	flt := float64(size) / 1000000000
	sizeToString := fmt.Sprintf("%.1f ГБ", flt)
	return sizeToString
}
