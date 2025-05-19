package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func EnableBackupThread(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				BackupNow()
				DeleteObsoleteBackups()
				time.Sleep(time.Duration(Config.BackupCfg.BackupInterval) * time.Minute)
			}
		}
	}()
}

func BackupNow() {
	// tar the blog directory and store it in the backup directory
	src := "configs/"
	dst := Config.BackupCfg.BackupDir + "/backup-" + time.Now().Format("2006-01-02_15-04-05") + ".tar.gz"
	// check dir
	if _, err := os.Stat(Config.BackupCfg.BackupDir); os.IsNotExist(err) {
		os.MkdirAll(Config.BackupCfg.BackupDir, 0755)
	}
	if err := TarDir(src, dst); err != nil {
		Log(3, "Error while backing up: "+err.Error())
	} else {
		Log(1, "Backup successful: "+dst)
	}
}

func DeleteObsoleteBackups() {
	files, err := filepath.Glob(Config.BackupCfg.BackupDir + "/backup-*.tar.gz")
	if err != nil {
		Log(3, "Error while deleting obsolete backups: "+err.Error())
		return
	}

	type backupFile struct {
		path string
		time time.Time
	}

	var backups []backupFile

	// 解析文件名中的时间信息
	for _, file := range files {
		fileName := filepath.Base(file)
		timeStr := strings.TrimSuffix(strings.TrimPrefix(fileName, "backup-"), ".tar.gz")

		t, err := time.Parse("2006-01-02_15-04-05", timeStr)
		if err != nil {
			Log(3, fmt.Sprintf("无法解析文件时间: %s, 错误: %v", fileName, err))
			continue
		}

		backups = append(backups, backupFile{path: file, time: t})
	}

	// 按备份时间从旧到新排序
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].time.Before(backups[j].time)
	})

	now := time.Now()
	var toDelete []string

	// 根据存活时间判断
	if Config.BackupCfg.MaxBackupsSurvivalTime > 0 {
		threshold := now.Add(time.Duration(-Config.BackupCfg.MaxBackupsSurvivalTime) * time.Minute)
		for _, b := range backups {
			if b.time.Before(threshold) {
				toDelete = append(toDelete, b.path)
			}
		}
	}

	// 根据最大备份数量判断（保留最新的N个）
	if Config.BackupCfg.MaxBackups > 0 && len(backups) > Config.BackupCfg.MaxBackups {
		preserve := len(backups) - Config.BackupCfg.MaxBackups
		for i := 0; i < preserve; i++ {
			toDelete = append(toDelete, backups[i].path)
		}
	}

	// 去重并删除文件
	seen := make(map[string]bool)
	for _, path := range toDelete {
		if !seen[path] {
			if err := os.Remove(path); err != nil {
				Log(3, fmt.Sprintf("删除备份失败: %s, 错误: %v", path, err))
			} else {
				Log(1, fmt.Sprintf("已删除过期备份: %s", path))
			}
			seen[path] = true
		}
	}
}

func TarDir(src string, dst string) (err error) {
	fw, err := os.Create(dst)
	if err != nil {
		return
	}
	defer fw.Close()

	gw := gzip.NewWriter(fw)
	defer gw.Close()

	tw := tar.NewWriter(gw)

	defer tw.Close()

	return filepath.Walk(src, func(fileName string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		hdr, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			return err
		}
		hdr.Name = strings.TrimPrefix(fileName, string(filepath.Separator))
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if !fi.Mode().IsRegular() {
			return nil
		}
		fr, err := os.Open(fileName)
		if err != nil {
			return err
		}
		defer fr.Close()
		_, err = io.Copy(tw, fr)
		if err != nil {
			return err
		}
		return nil
	})
}
