package util

import (
	"os"
	"path/filepath"
)

// SaveFile 保存文件
func SaveFile(path string, data []byte) error {
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// 写入文件
	return os.WriteFile(path, data, 0644)
}
