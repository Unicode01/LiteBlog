package main

import (
	utils "LiteBlog/utils"
	"context"
	"encoding/json"
	"os"
	"time"

	_ "embed"

	"github.com/fsnotify/fsnotify"
)

var (
	BackupThreadCancel context.CancelFunc
)

//go:embed configs/config.json
var DefaultConfig []byte

type AllConfig struct {
	ServerCfg         ServerConfig         `json:"server_config"`
	AccessCfg         AccessConfig         `json:"access_config"`
	CacheCfg          CacheConfig          `json:"cache_config"`
	DeliverCfg        DeliverConfig        `json:"deliver_config"`
	BackupCfg         BackupsConfig        `json:"backup_config"`
	CommentCfg        CommentConfig        `json:"comment_config"`
	LoggerCfg         utils.LoggerConfig   `json:"logger_config"`
	ContentAdvisorCfg ContentAdvisorConfig `json:"contentAdvisor_config"`
	NotifyCfg         NotifyConfig         `json:"notify_config"`
	PluginCfg         PluginConfig         `json:"plugins_config"`
	SnifferCfg        SnifferConfig        `json:"sniffer_config"`
	RenderCfg         RenderConfig         `json:"render_config"`
}

type ServerConfig struct {
	Host         string            `json:"host"`
	URLOrigin    string            `json:"url_origin"`
	Port         int               `json:"port"`
	TlsConfig    TlsConfig         `json:"tls_config"`
	ExtraHeaders map[string]string `json:"extra_headers"`
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
	RandomKey     bool   `json:"random_key"`
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
	Enabled                   bool   `json:"enabled"`
	Type                      string `json:"type"`
	CFSecretKey               string `json:"cf_secret_key"`
	CFSiteKey                 string `json:"cf_site_key"`
	GoogleSecretKey           string `json:"google_secret_key"`
	GoogleSiteKey             string `json:"google_site_key"`
	MinSecondsBetweenComments int    `json:"min_seconds_between_comments"`
	MaxTextLength             int    `json:"max_text_length"`
}

type ContentAdvisorConfig struct {
	Enabled       bool `json:"enabled"`
	FilterComment bool `json:"filter_comment"`
	FilterArticle bool `json:"filter_article"`
	FilterCard    bool `json:"filter_card"`
}

type NotifyConfig struct {
	Enabled    bool     `json:"enabled"`
	Type       string   `json:"type"`
	Trigger    []string `json:"trigger"`
	SMTPConfig struct {
		Host     string   `json:"host"`
		UserName string   `json:"username"`
		Password string   `json:"password"`
		FromAddr string   `json:"from_addr"`
		ToAddrs  []string `json:"to_addrs"`
	} `json:"smtp_config"`
	TelegramBotConfig struct {
		Token  string `json:"token"`
		ChatID string `json:"chat_id"`
	} `json:"telegrambot_config"`
}

type PluginConfig struct {
	Enabled bool `json:"enabled"`
	// gRPC 插件加载器配置
	GRPCConfig struct {
		Enabled         bool   `json:"enabled"`
		ListenerAddress string `json:"listener_address"`
		CommandTimeout  int    `json:"command_timeout"` // 命令超时时间（秒）
		AccessKey       string `json:"access_key"`      // 访问密钥，为空则不验证
	} `json:"grpc_config"`
	// JavaScript 插件加载器配置
	JSConfig struct {
		Enabled   bool   `json:"enabled"`
		PluginDir string `json:"plugin_dir"` // JS 插件目录
		InitDelay int    `json:"init_delay"` // 初始化延迟（秒）
	} `json:"js_config"`
	// 路由监听器配置
	RouteListenerConfig struct {
		MaxRequestBodySize  int64 `json:"max_request_body_size"`  // 最大请求体大小（字节），0 表示不限制，默认 10MB
		MaxResponseBodySize int64 `json:"max_response_body_size"` // 最大响应体大小（字节），0 表示不限制，默认 10MB
		CaptureRequestBody  bool  `json:"capture_request_body"`   // 是否捕获请求体，默认 true
		CaptureResponseBody bool  `json:"capture_response_body"`  // 是否捕获响应体，默认 true
	} `json:"route_listener_config"`
}

type SnifferConfig struct {
	Enabled        bool   `json:"enabled"`
	PublicProvider string `json:"public_provider"`
}

type RenderConfig struct {
	Render struct {
		RssFeed bool `json:"rss_feed"`
		SiteMap bool `json:"site_map"`
	} `json:"render"`
	MinRenderInterval int `json:"min_render_interval"`
	MaxRenderInterval int `json:"max_render_interval"`
}

func ReadConfig() AllConfig {
	configFile, err := os.ReadFile("configs/config.json")
	if err != nil {
		panic(err)
	}
	var config AllConfig
	json.Unmarshal(DefaultConfig, &config) // load default config
	err = json.Unmarshal(configFile, &config)
	if err != nil {
		panic(err)
	}
	utils.LoggerCfg = config.LoggerCfg
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
		utils.Log(1, "Config file(configs/config.json) changed, reloading...")
		Config = ReadConfig()
		BackupConfigures()
		SetVarToGlobalMap()
	})
	if err != nil {
		utils.Log(3, "Config watcher error:"+err.Error())
	}
	err = AddConfigListener("configs/global.json", func() {
		utils.Log(1, "Global file(configs/global.json) changed, reloading...")
		ReadGolbalConfig()
	})
	if err != nil {
		utils.Log(3, "Config watcher error:"+err.Error())
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
				utils.Log(3, "Config watcher error:"+err.Error())
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
			ctx, cancel := context.WithCancel(context.Background())
			EnableBackupThread(ctx)
			BackupThreadCancel = cancel
		}
	}
}

func SetVarToGlobalMap() {
	if Config.CommentCfg.Enabled {
		switch Config.CommentCfg.Type {
		case "cloudflare_turnstile":
			GlobalMapLocker.Lock()
			GlobalMap["cf_site_key"] = []byte(Config.CommentCfg.CFSiteKey)
			GlobalMap["comment_check_type"] = []byte(Config.CommentCfg.Type)
			GlobalMapLocker.Unlock()
		case "google_recaptcha":
			GlobalMapLocker.Lock()
			GlobalMap["google_site_key"] = []byte(Config.CommentCfg.GoogleSiteKey)
			GlobalMap["comment_check_type"] = []byte(Config.CommentCfg.Type)
			GlobalMapLocker.Unlock()
		}
	}
}
