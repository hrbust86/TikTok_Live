package main

import (
	"encoding/json"
	"log"
	"os"
	"sync"
)

// Config 配置结构体
type Config struct {
	WebSocketServer struct {
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
	} `json:"websocket_server"`

	TikTokProxy struct {
		Port          int    `json:"port"`
		UpstreamProxy string `json:"upstream_proxy"`
		Timeout       int    `json:"timeout"`
	} `json:"tiktok_proxy"`
}

var (
	config     *Config
	configOnce sync.Once
	configPath = "config.json"
)

// LoadConfig 加载配置文件
func LoadConfig() (*Config, error) {
	var loadErr error
	configOnce.Do(func() {
		config = &Config{}

		// 尝试从文件加载配置
		if err := loadConfigFromFile(); err != nil {
			log.Printf("加载配置文件失败: %v", err)
			loadErr = err
			return
		}
	})

	if loadErr != nil {
		return nil, loadErr
	}
	return config, nil
}

// setDefaultConfig 设置默认配置
func setDefaultConfig() {
	config.WebSocketServer.URL = "ws://127.0.0.1:18080/ws"
	config.WebSocketServer.Headers = map[string]string{
		"User-Agent": "TikTok-Live-Barrage-Client/1.0",
		"Origin":     "http://localhost",
	}

	config.TikTokProxy.Port = 23809
	config.TikTokProxy.UpstreamProxy = "socket5://127.0.0.1:21586"
	config.TikTokProxy.Timeout = 60000
}

// loadConfigFromFile 从文件加载配置
func loadConfigFromFile() error {
	file, err := os.Open(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(config)
}

// SaveConfig 保存配置到文件
func SaveConfig() error {
	file, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(config)
}

// UpdateWebSocketServer 更新WebSocket服务器配置
func UpdateWebSocketServer(url string, headers map[string]string) {
	config.WebSocketServer.URL = url
	if headers != nil {
		config.WebSocketServer.Headers = headers
	}

	// 保存配置
	if err := SaveConfig(); err != nil {
		log.Printf("保存配置失败: %v", err)
	}

	// 重新初始化WebSocket客户端
	initWebSocketClient()
}

// GetWebSocketServerURL 获取WebSocket服务器URL
func GetWebSocketServerURL() string {
	cfg, err := LoadConfig()
	if err != nil {
		log.Printf("获取WebSocket服务器URL失败: %v", err)
		return ""
	}
	return cfg.WebSocketServer.URL
}

// GetWebSocketServerHeaders 获取WebSocket服务器请求头
func GetWebSocketServerHeaders() map[string]string {
	cfg, err := LoadConfig()
	if err != nil {
		log.Printf("获取WebSocket服务器请求头失败: %v", err)
		return nil
	}
	return cfg.WebSocketServer.Headers
}

// GetTikTokProxyPort 获取TikTok代理端口
func GetTikTokProxyPort() int {
	cfg, err := LoadConfig()
	if err != nil {
		log.Printf("获取TikTok代理端口失败: %v", err)
		return 0
	}
	return cfg.TikTokProxy.Port
}

// GetUpstreamProxy 获取上游代理地址
func GetUpstreamProxy() string {
	cfg, err := LoadConfig()
	if err != nil {
		log.Printf("获取上游代理地址失败: %v", err)
		return ""
	}
	return cfg.TikTokProxy.UpstreamProxy
}

// GetProxyTimeout 获取代理超时时间
func GetProxyTimeout() int {
	cfg, err := LoadConfig()
	if err != nil {
		log.Printf("获取代理超时时间失败: %v", err)
		return 0
	}
	return cfg.TikTokProxy.Timeout
}
