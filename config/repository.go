package config

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
)

// Структура для манифеста Docker v2.2
type Manifest struct {
	SchemaVersion int               `json:"schemaVersion"` // Версия схемы (всегда 2)
	MediaType     string            `json:"mediaType"`     // Тип медиа для манифеста ("application/vnd.docker.distribution.manifest.v2+json")
	Config        *BlobDescriptor   `json:"config"`        // Информация о конфигурационном файле
	Layers        []*BlobDescriptor `json:"layers"`        // Массив слоёв
}

// BlobDescriptor описывает конфигурационный файл или слой
type BlobDescriptor struct {
	MediaType string `json:"mediaType"` // Тип медиа (например, "application/vnd.docker.container.image.v1+json" для конфига)
	Size      int64  `json:"size"`      // Размер объекта в байтах
	Digest    string `json:"digest"`    // Контрольная сумма (SHA-256)
}

// CalculateSizeAndDigest рассчитывает размер и контрольную сумму для файла
func CalculateSizeAndDigest(filename string) (int64, string, error) {
	// Чтение файла
	data, err := os.ReadFile(filename)
	if err != nil {
		return 0, "", fmt.Errorf("couldn't read file: %v", err)
	}

	size := int64(len(data))                                // Размер файла в байтах
	digest := fmt.Sprintf("sha256:%x", sha256.Sum256(data)) // Контрольная сумма SHA-256

	return size, digest, nil
}

// NewManifest создаёт новый экземпляр структуры Manifest
func NewManifest() *Manifest {
	return &Manifest{
		SchemaVersion: 2,
		MediaType:     "application/vnd.docker.distribution.manifest.v2+json",
		Config:        nil,
		Layers:        make([]*BlobDescriptor, 0),
	}
}

// AddConfig добавляет информацию о конфигурационном файле в манифест
func (m *Manifest) AddConfig(filename string) error {
	size, digest, err := CalculateSizeAndDigest(filename)
	if err != nil {
		return err
	}
	m.Config = &BlobDescriptor{
		MediaType: "application/vnd.docker.container.image.v1+json",
		Size:      size,
		Digest:    digest,
	}
	return nil
}

// AddLayer добавляет информацию о слое в манифест
func (m *Manifest) AddLayer(filename string) error {
	size, digest, err := CalculateSizeAndDigest(filename)
	if err != nil {
		return err
	}
	m.Layers = append(m.Layers, &BlobDescriptor{
		MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
		Size:      size,
		Digest:    digest,
	})
	return nil
}

// Print выводит манифест в формате JSON
func (m *Manifest) Print() error {
	data, err := m.MarshalJSON()
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

// MarshalJSON преобразует структуру в JSON
func (m *Manifest) MarshalJSON() ([]byte, error) {
	return json.Marshal(m)
}
