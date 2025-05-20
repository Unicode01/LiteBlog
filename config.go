package main

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
)

var (
	BackupThreadCancel context.CancelFunc
)

type AllConfig struct {
	ServerCfg  ServerConfig  `json:"server_config"`
	AccessCfg  AccessConfig  `json:"access_config"`
	CacheCfg   CacheConfig   `json:"cache_config"`
	DeliverCfg DeliverConfig `json:"deliver_config"`
	BackupCfg  BackupsConfig `json:"backup_config"`
	CommentCfg CommentConfig `json:"comment_config"`
}

type ServerConfig struct {
	Host      string    `json:"host"`
	Port      int       `json:"port"`
	TlsConfig TlsConfig `json:"tls_config"`
}

type TlsConfig struct {
	Enabled  bool   `json:"enabled"`
	CertFile string `json:"cert_file"`
	KeyFile  string `json:"key_file"`
}

type AccessConfig struct {
	EnableBackend bool   `json:"enable_backend"`
	BackendPath   string `json:"backend_path"`
	AccessToken   string `json:"access_token"`
}

type CacheConfig struct {
	UseDisk       bool  `json:"use_disk"`
	MaxCacheSize  int64 `json:"max_cache_size"`
	MaxCacheItems int   `json:"max_cache_items"`
	ExpireTime    int64 `json:"expire_time"`
}

type DeliverConfig struct {
	Buffer  int `json:"buffer"`
	Threads int `json:"threads"`
}

type BackupsConfig struct {
	Enabled                bool   `json:"enabled"`
	BackupDir              string `json:"backup_dir"`
	BackupInterval         int    `json:"backup_interval"`
	MaxBackups             int    `json:"max_backups"`
	MaxBackupsSurvivalTime int    `json:"max_backups_survival_time"`
}

type CommentConfig struct {
	Enable                    bool   `json:"enable"`
	Type                      string `json:"type"`
	CFSecretKey               string `json:"cf_secret_key"`
	CFSiteKey                 string `json:"cf_site_key"`
	MinSecondsBetweenComments int    `json:"min_seconds_between_comments"`
}

func ReadConfig() AllConfig {
	configFile, err := os.ReadFile("configs/config.json")
	if err != nil {
		panic(err)
	}
	var config AllConfig
	err = json.Unmarshal(configFile, &config)
	if err != nil {
		panic(err)
	}
	return config
}

func ReadGolbalConfig() {
	configFile, err := os.ReadFile("configs/global.json")
	if err != nil {
		panic(err)
	}
	var globMap map[string]interface{}
	json.Unmarshal(configFile, &globMap)
	for k, v := range globMap {
		vString, ok := v.(string)
		if ok {
			GlobalMapLocker.Lock()
			GlobalMap[k] = []byte(vString)
			GlobalMapLocker.Unlock()
		}
	}
}

func AutoAddListener() {
	err := AddConfigListener("configs/config.json", func() {
		Log(1, "Config file(configs/config.json) changed, reloading...")
		Config = ReadConfig()
		BackupConfigures()
		SetVarToGlobalMap()
	})
	if err != nil {
		Log(3, "Config watcher error:"+err.Error())
	}
	err = AddConfigListener("configs/global.json", func() {
		Log(1, "Global file(configs/global.json) changed, reloading...")
		ReadGolbalConfig()
	})
	if err != nil {
		Log(3, "Config watcher error:"+err.Error())
	}
}

func AddConfigListener(filePath string, function func()) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	err = watcher.Add(filePath)
	if err != nil {
		return err

	}

	var (
		debounceDuration = 500 * time.Millisecond // Anti-flapping debounce duration
		timer            *time.Timer
	)

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					// Anti-flapping：cancel the previous timer if there is one
					if timer != nil {
						timer.Stop()
					}
					timer = time.AfterFunc(debounceDuration, func() {
						// call the function after the debounce duration
						function()
					})
				}
			case err := <-watcher.Errors:
				Log(3, "Config watcher error:"+err.Error())
				return
			}
		}
	}()
	return nil
}

func BackupConfigures() {
	if !Config.BackupCfg.Enabled {
		if BackupThreadCancel != nil {
			BackupThreadCancel()
			BackupThreadCancel = nil
		}
		return
	} else {
		if BackupThreadCancel == nil {
			ctx, cancle := context.WithCancel(context.Background())
			EnableBackupThread(ctx)
			BackupThreadCancel = cancle
		}
	}
}

func SetVarToGlobalMap() {
	if Config.CommentCfg.Enable {
		switch Config.CommentCfg.Type {
		case "cloudflare_turnstile":
			GlobalMapLocker.Lock()
			GlobalMap["cf_site_key"] = []byte(Config.CommentCfg.CFSiteKey)
			GlobalMap["comment_check_type"] = []byte(Config.CommentCfg.Type)
			GlobalMapLocker.Unlock()
		}
	}
}
