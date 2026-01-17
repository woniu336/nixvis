package util

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	globalConfig *Config
	websiteIDMap sync.Map
)

type Config struct {
	System   SystemConfig    `json:"system"`
	Server   ServerConfig    `json:"server"`
	Websites []WebsiteConfig `json:"websites"`
	PVFilter PVFilterConfig  `json:"pvFilter"`
}

type WebsiteConfig struct {
	Name    string `json:"name"`
	LogPath string `json:"logPath"`
}

type SystemConfig struct {
	LogDestination string `json:"logDestination"`
	TaskInterval   string `json:"taskInterval"` // "5m" "25s"
}

type ServerConfig struct {
	Port string `json:"Port"`
}

type PVFilterConfig struct {
	StatusCodeInclude []int    `json:"statusCodeInclude"`
	ExcludePatterns   []string `json:"excludePatterns"`
	ExcludeIPs        []string `json:"excludeIPs"`
}

// ReadRawConfig 读取配置文件但不初始化全局变量
func ReadRawConfig() (*Config, error) {
	// 读取文件内容
	bytes, err := os.ReadFile(ConfigFile)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	err = json.Unmarshal(bytes, cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// ReadConfig 读取配置文件并返回配置，同时初始化 ID 映射
func ReadConfig() *Config {
	if globalConfig != nil {
		return globalConfig
	}

	// 读取文件内容
	bytes, err := os.ReadFile(ConfigFile)
	if err != nil {
		panic(err)
	}

	cfg := &Config{}
	err = json.Unmarshal(bytes, cfg)
	if err != nil {
		panic(err)
	}

	// 初始化 ID 映射
	for _, website := range cfg.Websites {
		id := generateID(website.Name)
		websiteIDMap.Store(id, website)
	}

	globalConfig = cfg
	return globalConfig
}

// GetWebsiteByID 根据 ID 获取对应的 WebsiteConfig
func GetWebsiteByID(id string) (WebsiteConfig, bool) {
	value, ok := websiteIDMap.Load(id)
	if ok {
		return value.(WebsiteConfig), true
	}
	return WebsiteConfig{}, false
}

// GetAllWebsiteIDs 获取所有网站的 ID 列表
func GetAllWebsiteIDs() []string {
	var ids []string
	websiteIDMap.Range(func(key, value interface{}) bool {
		ids = append(ids, key.(string))
		return true
	})
	return ids
}

// ParseInterval 解析间隔配置字符串，支持分钟(m)和秒(s)单位
func ParseInterval(intervalStr string, defaultInterval time.Duration) time.Duration {
	if intervalStr == "" {
		return defaultInterval
	}

	// 尝试解析配置的时间间隔
	duration, err := time.ParseDuration(intervalStr)
	if err != nil {
		logrus.WithField("interval", intervalStr).Info(
			"无效的时间间隔配置，使用默认值")
		return defaultInterval
	}

	minInterval := 5 * time.Second
	if duration < minInterval {
		logrus.WithField("interval", intervalStr).Info(
			"配置的时间间隔过短，已调整为最小值5秒")
		return minInterval
	}

	return duration
}

// generateID 根据输入字符串生成唯一 ID
func generateID(input string) string {
	hash := md5.Sum([]byte(input))
	return hex.EncodeToString(hash[:2])
}

// GenerateID 导出的ID生成函数，供外部使用
func GenerateID(input string) string {
	return generateID(input)
}

// SaveConfig 保存配置到文件
func SaveConfig(cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(ConfigFile, data, 0644)
}

// AddWebsite 添加站点
func AddWebsite(name, logPath string) error {
	cfg, err := ReadRawConfig()
	if err != nil {
		return err
	}

	// 检查是否已存在同名站点
	for _, site := range cfg.Websites {
		if site.Name == name {
			return fmt.Errorf("站点 %s 已存在", name)
		}
	}

	cfg.Websites = append(cfg.Websites, WebsiteConfig{
		Name:    name,
		LogPath: logPath,
	})

	return SaveConfig(cfg)
}

// RemoveWebsite 删除站点
func RemoveWebsite(id string) error {
	cfg, err := ReadRawConfig()
	if err != nil {
		return err
	}

	// 查找并删除站点
	website, ok := GetWebsiteByID(id)
	if !ok {
		return fmt.Errorf("站点 %s 不存在", id)
	}

	newWebsites := make([]WebsiteConfig, 0, len(cfg.Websites))
	for _, site := range cfg.Websites {
		if site.Name != website.Name {
			newWebsites = append(newWebsites, site)
		}
	}

	cfg.Websites = newWebsites
	if err := SaveConfig(cfg); err != nil {
		return err
	}

	// 从映射中删除
	websiteIDMap.Delete(id)

	return nil
}

// ReloadConfig 重新加载配置并更新映射
func ReloadConfig() error {
	// 读取配置文件
	bytes, err := os.ReadFile(ConfigFile)
	if err != nil {
		return err
	}

	cfg := &Config{}
	err = json.Unmarshal(bytes, cfg)
	if err != nil {
		return err
	}

	// 清空并重新初始化 ID 映射
	websiteIDMap = sync.Map{}
	for _, website := range cfg.Websites {
		id := generateID(website.Name)
		websiteIDMap.Store(id, website)
	}

	globalConfig = cfg
	return nil
}

// ResetConfigCache 重置全局配置缓存
func ResetConfigCache() {
	globalConfig = nil
	websiteIDMap = sync.Map{}
}

// GetPVFilterConfig 获取PV过滤配置
func GetPVFilterConfig() PVFilterConfig {
	cfg := ReadConfig()
	if cfg.PVFilter.ExcludePatterns == nil {
		cfg.PVFilter.ExcludePatterns = []string{}
	}
	if cfg.PVFilter.ExcludeIPs == nil {
		cfg.PVFilter.ExcludeIPs = []string{}
	}
	return cfg.PVFilter
}

// AddExcludePattern 添加排除模式
func AddExcludePattern(pattern string) error {
	cfg, err := ReadRawConfig()
	if err != nil {
		return err
	}

	if cfg.PVFilter.ExcludePatterns == nil {
		cfg.PVFilter.ExcludePatterns = []string{}
	}

	// 检查是否已存在
	for _, p := range cfg.PVFilter.ExcludePatterns {
		if p == pattern {
			return fmt.Errorf("排除模式 %s 已存在", pattern)
		}
	}

	cfg.PVFilter.ExcludePatterns = append(cfg.PVFilter.ExcludePatterns, pattern)
	return SaveConfig(cfg)
}

// RemoveExcludePattern 移除排除模式
func RemoveExcludePattern(pattern string) error {
	cfg, err := ReadRawConfig()
	if err != nil {
		return err
	}

	if cfg.PVFilter.ExcludePatterns == nil {
		return fmt.Errorf("排除模式 %s 不存在", pattern)
	}

	// 查找并删除
	found := false
	newPatterns := make([]string, 0, len(cfg.PVFilter.ExcludePatterns))
	for _, p := range cfg.PVFilter.ExcludePatterns {
		if p != pattern {
			newPatterns = append(newPatterns, p)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("排除模式 %s 不存在", pattern)
	}

	cfg.PVFilter.ExcludePatterns = newPatterns
	return SaveConfig(cfg)
}

// AddExcludeIP 添加排除IP
func AddExcludeIP(ip string) error {
	cfg, err := ReadRawConfig()
	if err != nil {
		return err
	}

	if cfg.PVFilter.ExcludeIPs == nil {
		cfg.PVFilter.ExcludeIPs = []string{}
	}

	// 检查是否已存在
	for _, i := range cfg.PVFilter.ExcludeIPs {
		if i == ip {
			return fmt.Errorf("排除IP %s 已存在", ip)
		}
	}

	cfg.PVFilter.ExcludeIPs = append(cfg.PVFilter.ExcludeIPs, ip)
	return SaveConfig(cfg)
}

// RemoveExcludeIP 移除排除IP
func RemoveExcludeIP(ip string) error {
	cfg, err := ReadRawConfig()
	if err != nil {
		return err
	}

	if cfg.PVFilter.ExcludeIPs == nil {
		return fmt.Errorf("排除IP %s 不存在", ip)
	}

	// 查找并删除
	found := false
	newIPs := make([]string, 0, len(cfg.PVFilter.ExcludeIPs))
	for _, i := range cfg.PVFilter.ExcludeIPs {
		if i != ip {
			newIPs = append(newIPs, i)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("排除IP %s 不存在", ip)
	}

	cfg.PVFilter.ExcludeIPs = newIPs
	return SaveConfig(cfg)
}
