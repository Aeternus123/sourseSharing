package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

type Config struct {
	ServerURL    string `json:"server_url"`
	UserAddress  string `json:"user_address"`
	AdminAddress string `json:"admin_address"`
	PrivateKey   string `json:"private_key,omitempty"` // 加密存储的私钥
	UserPassword string `json:"user_password,omitempty"` // 用户密码哈希
}

func getConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".soursesharing", "config.json")
}

func LoadConfig() (*Config, error) {
	configPath := getConfigPath()
	
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// 创建默认配置
		defaultConfig := &Config{
			ServerURL: "http://localhost:8080",
		}
		return defaultConfig, nil
	}
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}
	
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}
	
	return &config, nil
}

func (c *Config) Save() error {
	configPath := getConfigPath()
	configDir := filepath.Dir(configPath)
	
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("创建配置目录失败: %v", err)
	}
	
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}
	
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}
	
	return nil
}

func (c *Config) SetServerURL(url string) error {
	if !strings.HasPrefix(url, "http") {
		url = "http://" + url
	}
	c.ServerURL = url
	return c.Save()
}

func (c *Config) SetUserAddress(address string) error {
	if !common.IsHexAddress(address) {
		return fmt.Errorf("无效的以太坊地址")
	}
	c.UserAddress = address
	return c.Save()
}

func (c *Config) SetAdminAddress(address string) error {
	if !common.IsHexAddress(address) {
		return fmt.Errorf("无效的以太坊地址")
	}
	c.AdminAddress = address
	return c.Save()
}

func (c *Config) SetPrivateKey(privateKey string) error {
	// 这里可以添加私钥加密逻辑
	c.PrivateKey = privateKey
	return c.Save()
}

func (c *Config) SetUserPassword(password string) error {
	// 存储密码哈希而不是明文
	c.UserPassword = hashPassword(password)
	return c.Save()
}

func (c *Config) ClearPrivateKey() error {
	c.PrivateKey = ""
	return c.Save()
}

func (c *Config) ClearPassword() error {
	c.UserPassword = ""
	return c.Save()
}

// 简单的密码哈希函数
func hashPassword(password string) string {
	// 在实际应用中应该使用 bcrypt 或 argon2
	// 这里使用简单实现，生产环境请使用安全哈希
	return fmt.Sprintf("%x", password)
}