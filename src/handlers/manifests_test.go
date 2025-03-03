package handlers

import (
	"os"
	"testing"
)

func TestWriteManigest(t *testing.T) {
	dataFile := "test.txt"
	text := []byte("hello world")
	t.Run("запись данных во временный файл", func(t *testing.T) {
		err := os.WriteFile(dataFile, text, 0755)
		if err != nil {
			t.Log(err)
		}
	})
	t.Run("чтение файла", func(t *testing.T) {
		content, err := os.ReadFile(dataFile)
		if err != nil {
			t.Error(err)
		}
		t.Log(string(content))
		os.Remove(dataFile)
	})
}
