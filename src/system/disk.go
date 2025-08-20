package system

import (
	"syscall"
)

type Disk struct {
	Total uint64
	Free  uint64
}

func DiskUsage() (Disk, error) {
	fs := syscall.Statfs_t{}
	err := syscall.Statfs("/", &fs)
	if err != nil {
		return Disk{}, err
	}

	blockSize := uint64(fs.Bsize) // Размер блока в байтах
	totalBlocks := fs.Blocks      // Всего блоков
	freeBlocks := fs.Bavail       // Доступных блоков для обычного пользователя

	totalBytes := blockSize * totalBlocks
	freeBytes := blockSize * freeBlocks
	return Disk{Total: totalBytes, Free: freeBytes}, nil
}
