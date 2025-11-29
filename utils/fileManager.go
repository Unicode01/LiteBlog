package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileMetadata 文件元数据结构
type FileMetadata struct {
	Filename     string `json:"filename"`
	OriginalName string `json:"original_name"`
	Size         int64  `json:"size"`
	UploadTime   int64  `json:"upload_time"`
	ExpiryTime   int64  `json:"expiry_time"` // 0 表示永不过期
	Hash         string `json:"hash"`
}

var (
	fileMetadataCache = make(map[string]FileMetadata) // filename -> metadata
	fileMetadataMutex sync.RWMutex
	timeWheel         *TimeWheel
)

// InitFileManager 初始化文件管理器
func InitFileManager(ctx context.Context) error {
	// 创建元数据目录
	if err := os.MkdirAll("upload/.metadata", 0755); err != nil {
		return fmt.Errorf("failed to create metadata directory: %w", err)
	}

	// 加载现有元数据
	if err := loadAllMetadata(); err != nil {
		Log(2, fmt.Sprintf("Failed to load metadata: %s", err))
	}

	// 初始化时间轮（每小时检查一次）
	timeWheel = NewTimeWheel(time.Hour, 24, ctx)
	timeWheel.Start()

	// 启动清理任务
	go startFileCleanupTask(ctx)

	Log(1, "File manager initialized")
	return nil
}

// SaveFileMetadata 保存文件元数据
func SaveFileMetadata(filename string, metadata FileMetadata) error {
	fileMetadataMutex.Lock()
	defer fileMetadataMutex.Unlock()

	// 保存到缓存
	fileMetadataCache[filename] = metadata

	// 检测 metadata 目录是否存在
	metadataDir := filepath.Join("upload", ".metadata")
	if _, err := os.Stat(metadataDir); os.IsNotExist(err) {
		err := os.MkdirAll(metadataDir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create metadata directory: %w", err)
		}
	}
	// 保存到文件
	metadataPath := filepath.Join(metadataDir, filename+".json")
	file, err := os.Create(metadataPath)
	if err != nil {
		return fmt.Errorf("failed to create metadata file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(metadata); err != nil {
		return fmt.Errorf("failed to encode metadata: %w", err)
	}

	return nil
}

// GetFileMetadata 获取文件元数据
func GetFileMetadata(filename string) (FileMetadata, bool) {
	fileMetadataMutex.RLock()
	defer fileMetadataMutex.RUnlock()

	metadata, exists := fileMetadataCache[filename]
	return metadata, exists
}

// deleteFileMetadata 删除文件元数据
func deleteFileMetadata(filename string) error {
	fileMetadataMutex.Lock()
	defer fileMetadataMutex.Unlock()

	// 从缓存删除
	delete(fileMetadataCache, filename)

	// 删除元数据文件
	metadataPath := filepath.Join("upload", ".metadata", filename+".json")
	if err := os.Remove(metadataPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete metadata file: %w", err)
	}

	return nil
}

// loadAllMetadata 加载所有元数据
func loadAllMetadata() error {
	metadataDir := filepath.Join("upload", ".metadata")

	// 检查目录是否存在
	if _, err := os.Stat(metadataDir); os.IsNotExist(err) {
		return nil
	}

	entries, err := os.ReadDir(metadataDir)
	if err != nil {
		return fmt.Errorf("failed to read metadata directory: %w", err)
	}

	fileMetadataMutex.Lock()
	defer fileMetadataMutex.Unlock()

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		metadataPath := filepath.Join(metadataDir, entry.Name())
		file, err := os.Open(metadataPath)
		if err != nil {
			Log(2, fmt.Sprintf("Failed to open metadata file %s: %s", entry.Name(), err))
			continue
		}

		var metadata FileMetadata
		decoder := json.NewDecoder(file)
		if err := decoder.Decode(&metadata); err != nil {
			Log(2, fmt.Sprintf("Failed to decode metadata file %s: %s", entry.Name(), err))
			file.Close()
			continue
		}
		file.Close()

		fileMetadataCache[metadata.Filename] = metadata
	}

	Log(1, fmt.Sprintf("Loaded %d file metadata entries", len(fileMetadataCache)))
	return nil
}

// startFileCleanupTask 启动文件清理任务
func startFileCleanupTask(ctx context.Context) {
	// 立即执行一次清理
	cleanupExpiredFiles()

	// 每小时执行一次清理
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			Log(1, "File cleanup task stopped")
			return
		case <-ticker.C:
			cleanupExpiredFiles()
		}
	}
}

// cleanupExpiredFiles 清理过期文件
func cleanupExpiredFiles() {
	fileMetadataMutex.RLock()
	expiredFiles := make([]string, 0)
	currentTime := time.Now().Unix()

	for filename, metadata := range fileMetadataCache {
		// ExpiryTime == 0 表示永不过期
		if metadata.ExpiryTime > 0 && metadata.ExpiryTime < currentTime {
			expiredFiles = append(expiredFiles, filename)
		}
	}
	fileMetadataMutex.RUnlock()

	// 删除过期文件
	for _, filename := range expiredFiles {
		if err := DeleteUploadedFile(filename); err != nil {
			Log(2, fmt.Sprintf("Failed to delete expired file %s: %s", filename, err))
		} else {
			Log(1, fmt.Sprintf("Deleted expired file: %s", filename))
		}
	}

	if len(expiredFiles) > 0 {
		Log(1, fmt.Sprintf("Cleaned up %d expired files", len(expiredFiles)))
	}
}

// DeleteUploadedFile 删除上传的文件及其元数据
func DeleteUploadedFile(filename string) error {
	// 删除文件
	filePath := filepath.Join("upload", filename)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	// 删除元数据
	if err := deleteFileMetadata(filename); err != nil {
		return fmt.Errorf("failed to delete metadata: %w", err)
	}

	return nil
}

// GetUploadStats 获取上传文件统计信息
func GetUploadStats() map[string]interface{} {
	fileMetadataMutex.RLock()
	defer fileMetadataMutex.RUnlock()

	totalSize := int64(0)
	totalFiles := len(fileMetadataCache)
	permanentFiles := 0
	expiredFiles := 0
	currentTime := time.Now().Unix()

	for _, metadata := range fileMetadataCache {
		totalSize += metadata.Size
		if metadata.ExpiryTime == 0 {
			permanentFiles++
		} else if metadata.ExpiryTime < currentTime {
			expiredFiles++
		}
	}

	return map[string]interface{}{
		"total_files":     totalFiles,
		"total_size":      totalSize,
		"permanent_files": permanentFiles,
		"expired_files":   expiredFiles,
	}
}
