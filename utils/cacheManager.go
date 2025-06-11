package utils

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type CacheManager struct {
	cacheDirectory string
}

var (
	ErrCacheExpired = fmt.Errorf("cache item is expired")
)

func NewCacheManager(maxCacheSize int64, maxCacheItems int) *CacheManager {
	cm := &CacheManager{
		cacheDirectory: "cache/",
	}
	// chech cache directory exists or not
	if _, err := os.Stat(cm.cacheDirectory); os.IsNotExist(err) {
		err := os.Mkdir(cm.cacheDirectory, 0755)
		if err != nil {
			return nil
		}
	}
	cm.ClearAllCache()

	go cm.autoCleanCache(maxCacheSize, maxCacheItems)
	return cm
}

// write cache item to file
// if cache item already exists, it will rewrites it
func (cm *CacheManager) AddCacheItem(CacheKey string, reader io.Reader, timeoutSecs int64) error {
	CacheKey = keysha256(CacheKey)
	// check if cache item already exists
	cacheFilePath := cm.cacheDirectory + CacheKey + ".cache"
	if _, err := os.Stat(cacheFilePath); !os.IsNotExist(err) { // cache item already exists, delete it first
		// delete old cache item
		err = os.Remove(cacheFilePath)
		if err != nil {
			return err
		}
	} else if _, err := os.Stat(cm.cacheDirectory); os.IsNotExist(err) { // if cache directory not exists, create it
		err := os.Mkdir(cm.cacheDirectory, 0755)
		if err != nil {
			return err
		}
	}
	// create cache file
	f, err := os.Create(cacheFilePath)
	if err != nil {
		return err
	}
	defer f.Close()
	TimeoutStamp := time.Now().Unix() + timeoutSecs
	// write cache timeout stamp to file
	timestampBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(timestampBytes, uint32(TimeoutStamp))
	_, err = f.Write(timestampBytes)
	if err != nil {
		return err
	}
	// write cache item to file
	_, err = io.Copy(f, reader)
	if err != nil {
		return err
	}
	return nil
}

func (cm *CacheManager) GetCacheItem(CacheKey string) (*os.File, error) {
	CacheKey = keysha256(CacheKey)
	// check if cache item exists
	cacheFilePath := cm.cacheDirectory + CacheKey + ".cache"
	if _, err := os.Stat(cacheFilePath); os.IsNotExist(err) {
		return nil, err
	}
	// open cache file
	f, err := os.Open(cacheFilePath)
	if err != nil {
		return nil, err
	}
	// read cache timeout stamp from file
	timestampBytes := make([]byte, 4)
	_, err = f.Read(timestampBytes)
	if err != nil {
		return nil, err
	}
	timeoutStamp := int64(binary.BigEndian.Uint32(timestampBytes))
	// check if cache item is expired
	if time.Now().Unix() > timeoutStamp {
		return nil, ErrCacheExpired
	}
	// read cache item from file
	f.Seek(4, io.SeekStart)
	return f, nil
}

func (cm *CacheManager) DelCacheItem(CacheKey string) error {
	CacheKey = keysha256(CacheKey)
	cacheFilePath := cm.cacheDirectory + CacheKey + ".cache"
	if _, err := os.Stat(cacheFilePath); os.IsNotExist(err) {
		return err
	}
	return os.Remove(cacheFilePath)
}

func (cm *CacheManager) autoCleanCache(maxCacheSize int64, maxCacheItems int) {
	for {
		cm.CleanCache(maxCacheSize, maxCacheItems)
		time.Sleep(time.Minute * 1)
	}
}

type cacheFileInfo struct {
	path    string
	size    int64
	modTime time.Time
}

func (cm *CacheManager) CleanCache(maxCacheSize int64, maxCacheItems int) error {
	cacheDir := cm.cacheDirectory
	files, err := os.ReadDir(cacheDir)
	if err != nil {
		return err
	}

	now := time.Now().Unix()
	var validFiles []cacheFileInfo
	totalSize := int64(0)

	// delete expired cache files and collect valid files
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(cacheDir, file.Name())
		f, err := os.Open(filePath)
		if err != nil {
			continue
		}

		timestampBytes := make([]byte, 4)
		_, err = f.Read(timestampBytes)
		f.Close()

		if err != nil {
			os.Remove(filePath)
			continue
		}

		timeoutStamp := int64(binary.BigEndian.Uint32(timestampBytes))
		if now > timeoutStamp {
			os.Remove(filePath)
			continue
		}

		// get info of cache file
		info, err := file.Info()
		if err != nil {
			continue
		}

		validFiles = append(validFiles, cacheFileInfo{
			path:    filePath,
			size:    info.Size(),
			modTime: info.ModTime(),
		})
		totalSize += info.Size()
	}

	// check if cache size is less than max cache size and cache items is less than max cache items
	totalItems := len(validFiles)
	if totalSize <= maxCacheSize && totalItems <= maxCacheItems {
		return nil
	}

	// sort valid files by modTime
	sort.Slice(validFiles, func(i, j int) bool {
		return validFiles[i].modTime.Before(validFiles[j].modTime)
	})

	// delete oldest cache files until cache size is less than max cache size and cache items is less than max cache items
	for _, file := range validFiles {
		if totalSize <= maxCacheSize && totalItems <= maxCacheItems {
			break
		}

		if err := os.Remove(file.path); err == nil {
			totalSize -= file.size
			totalItems--
		}
	}

	return nil
}

func (cm *CacheManager) ClearAllCache() error {
	cacheDir := cm.cacheDirectory
	files, err := os.ReadDir(cacheDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(cacheDir, file.Name())
		if err := os.Remove(filePath); err != nil {
			return err
		}
	}

	return nil
}

// tool functions

func keysha256(key string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
}
