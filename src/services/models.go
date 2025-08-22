package services

/*
Settings абстрактная структура для хранения настроек сервиса
*/
type Settings struct {
	Count         int
	Total         string
	Free          string
	FreeToPercent int
	Version       string
}
