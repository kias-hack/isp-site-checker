package config

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/kias-hack/isp-site-checker/internal/isp"
	"github.com/pelletier/go-toml"
)

type Config struct {
	MgrCtlPath     string `toml:"mgrctl_path"`
	DebugMode      bool
	ScrapeInterval time.Duration `toml:"scrape_interval"`

	Recipient string `toml:"recipient"`

	SMTP struct {
		Host     string `toml:"host"`
		Port     int    `toml:"port"`
		Username string `toml:"username"`
		Password string `toml:"password"`
		From     string `toml:"from"`
	}
}

func LoadConfig() *Config {
	slog.Info("start create config")

	cfg := &Config{}

	var configPath string

	flag.StringVar(&configPath, "config", "", "")
	flag.BoolVar(&cfg.DebugMode, "debug", false, "")

	flag.Parse()

	if configPath == "" {
		panic(fmt.Errorf("необходимо указать путь к конфигурационному файлу"))
	}

	bytes, err := os.ReadFile(configPath)
	if err != nil {
		panic(fmt.Errorf("ошибка чтения файла с конфигурацией: %s", err))
	}

	if err := toml.Unmarshal(bytes, cfg); err != nil {
		panic(fmt.Errorf("ошибка декодирования конфигурации: %s", err))
	}

	if cfg.ScrapeInterval.Seconds() == 0 {
		cfg.ScrapeInterval = time.Minute
	}

	if cfg.MgrCtlPath == "" {
		cfg.MgrCtlPath = isp.MGR_CTL_PATH_DEFAULT
	}

	if cfg.Recipient == "" {
		panic("получатель в настройках обязателен")
	}

	if cfg.SMTP.From == "" || cfg.SMTP.Host == "" || cfg.SMTP.Password == "" || cfg.SMTP.Port <= 0 || cfg.SMTP.Username == "" {
		panic("проверьте настройки SMTP")
	}

	return cfg
}
