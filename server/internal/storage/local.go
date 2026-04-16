// server/internal/storage/local.go
package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// LocalStorage 本地文件存储实现
type LocalStorage struct {
	basePath string // 本地存储根目录，如 ./uploads
	baseURL  string // 文件访问的 URL 前缀，如 http://localhost:8080/uploads
}

// NewLocalStorage 创建本地存储实例
// basePath: 本地存储根目录
// baseURL: 文件访问的 URL 前缀
func NewLocalStorage(basePath, baseURL string) *LocalStorage {
	// 确保存储根目录存在
	os.MkdirAll(basePath, 0755)
	return &LocalStorage{
		basePath: basePath,
		baseURL:  baseURL,
	}
}

// Upload 上传文件到本地磁盘
func (s *LocalStorage) Upload(ctx context.Context, file io.Reader, path string) (string, error) {
	fullPath := filepath.Join(s.basePath, path)

	// 确保目标目录存在
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// 创建目标文件
	dst, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	// 写入文件内容
	if _, err := io.Copy(dst, file); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	// 返回可访问的 URL
	url := s.baseURL + "/" + path
	return url, nil
}

// Download 从本地磁盘读取文件
func (s *LocalStorage) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.basePath, path)
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	return file, nil
}

// Delete 从本地磁盘删除文件
func (s *LocalStorage) Delete(ctx context.Context, path string) error {
	fullPath := filepath.Join(s.basePath, path)
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// GetURL 获取文件的访问 URL
func (s *LocalStorage) GetURL(ctx context.Context, path string) (string, error) {
	fullPath := filepath.Join(s.basePath, path)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return "", fmt.Errorf("file not found: %s", path)
	}
	return s.baseURL + "/" + path, nil
}
