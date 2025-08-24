package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port string `yaml:"port"`
	} `yaml:"server"`
	Emby Emby `yaml:"emby"`
}

type Emby struct {
	Host             string `yaml:"host"`
	APIKey           string `yaml:"api_key"`
	Strm             Strm   `yaml:"strm"`
	DownloadStrategy string `yaml:"download-strategy"`
}

type Strm struct {
	PathMap []string `yaml:"path-map"`
	pathMap [][2]string
}

func (e *Emby) Init() error {
	if strings.TrimSpace(e.Host) == "" {
		return fmt.Errorf("emby.host 配置不能为空")
	}
	if strings.TrimSpace(e.DownloadStrategy) == "" {
		e.DownloadStrategy = "403"
	}
	return e.Strm.Init()
}

func (s *Strm) Init() error {
	s.pathMap = make([][2]string, 0, len(s.PathMap))
	for _, rule := range s.PathMap {
		parts := strings.SplitN(rule, "=>", 2)
		if len(parts) != 2 {
			log.Printf("[WARN] 忽略无效的 path-map 规则: %s", rule)
			continue
		}
		from := strings.TrimSpace(parts[0])
		to := strings.TrimSpace(parts[1])
		if from != "" && to != "" {
			s.pathMap = append(s.pathMap, [2]string{from, to})
			log.Printf("[INFO] 加载路径映射规则: '%s' => '%s'", from, to)
		}
	}
	return nil
}

func (s *Strm) MapPath(path string) string {
	for _, m := range s.pathMap {
		from, to := m[0], m[1]
		if strings.Contains(path, from) {
			return strings.Replace(path, from, to, 1)
		}
	}
	return path
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败 %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	if strings.TrimSpace(cfg.Server.Port) == "" {
		return nil, fmt.Errorf("server.port 不能为空")
	}
	if err := cfg.Emby.Init(); err != nil {
		return nil, fmt.Errorf("Emby 配置初始化失败: %w", err)
	}

	return &cfg, nil
}