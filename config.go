package main

import (
	"encoding/json"
	"os"
)

type AllConfig struct {
	ServerCfg ServerConfig `json:"server_config"`
	AccessCfg AccessConfig `json:"access_config"`
	CacheCfg  CacheConfig  `json:"cache_config"`
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
	BackendPath string `json:"backend_path"`
	AccessToken string `json:"access_token"`
}

type CacheConfig struct {
	MaxCacheSize  int64 `json:"max_cache_size"`
	MaxCacheItems int   `json:"max_cache_items"`
	ExpireTime    int64 `json:"expire_time"`
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
	ReadGolbalConfig()
	return config
}

func ReadGolbalConfig() {
	configFile, err := os.ReadFile("configs/global.json")
	if err != nil {
		panic(err)
	}
	var globMap map[string]string
	json.Unmarshal(configFile, &globMap)
	for k, v := range globMap {
		GlobalMapLocker.Lock()
		GlobalMap[k] = []byte(v)
		GlobalMapLocker.Unlock()
	}
}
