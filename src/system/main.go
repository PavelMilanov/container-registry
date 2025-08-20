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

func HumanizeSize(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB",
		float64(bytes)/float64(div), "KMGTPE"[exp])
}
