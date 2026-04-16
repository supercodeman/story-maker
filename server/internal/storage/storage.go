// server/internal/storage/storage.go
package storage

import (
	"context"
	"io"
)

// Storage 文件存储抽象接口
// 遵循 ISP（接口隔离原则）：仅定义文件存储必需的四个操作
// 本期实现 LocalStorage，后续切换 OSS 只需替换实现（OCP 开闭原则）
type Storage interface {
	// Upload 上传文件，返回可访问的 URL 或路径
	Upload(ctx context.Context, file io.Reader, path string) (string, error)

	// Download 下载文件，返回文件内容的 ReadCloser
	Download(ctx context.Context, path string) (io.ReadCloser, error)

	// Delete 删除文件
	Delete(ctx context.Context, path string) error

	// GetURL 获取文件的访问 URL
	GetURL(ctx context.Context, path string) (string, error)
}
