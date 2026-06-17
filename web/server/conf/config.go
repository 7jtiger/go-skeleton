package conf

import (
	"fmt"
	"os"

	"github.com/naoina/toml"
)

type Config struct {
	Server ServerConfig `toml:"server"`
	Admin  AdminConfig  `toml:"admin"`
	Data   DataConfig   `toml:"data"`
}

type ServerConfig struct {
	Port      string `toml:"port"`
	Mode      string `toml:"mode"`
	StaticDir string `toml:"staticDir"`
}

type AdminConfig struct {
	Username     string `toml:"username"`
	Password     string `toml:"password"`
	JWTSecret    string `toml:"jwtSecret"`
	JWTExpireMin int    `toml:"jwtExpireMin"`
}

type DataConfig struct {
	ContentPath string `toml:"contentPath"`
	InquiryPath string `toml:"inquiryPath"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if cfg.Server.Port == "" {
		cfg.Server.Port = ":3000"
	}
	if cfg.Server.StaticDir == "" {
		cfg.Server.StaticDir = "../dist"
	}
	if cfg.Data.ContentPath == "" {
		cfg.Data.ContentPath = "./data/content.json"
	}
	if cfg.Data.InquiryPath == "" {
		cfg.Data.InquiryPath = "./data/inquiries.json"
	}
	if cfg.Admin.JWTExpireMin == 0 {
		cfg.Admin.JWTExpireMin = 480
	}

	return &cfg, nil
}
