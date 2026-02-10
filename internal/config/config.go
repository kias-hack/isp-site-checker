package config

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/kias-hack/isp-site-checker/internal/isp"
	"github.com/kias-hack/isp-site-checker/internal/util"
	"github.com/pelletier/go-toml"
)

type Config struct {
	MgrCtlPath            string `toml:"mgrctl_path"`
	DebugMode             bool
	ScrapeInterval        time.Duration `toml:"scrape_interval"`
	SiteRetentionInterval time.Duration `toml:"site_retention_interval"`

	SMTP struct {
		Host     string `toml:"host"`
		Port     string `toml:"port"`
		Username string `toml:"email"`
		Password string `toml:"password"`
		UseTLS   bool   `toml:"use_tls"`
	}

	EMail struct {
		From    string   `toml:"from"`
		To      []string `toml:"to"`
		Subject string   `toml:"subject"`
	}

	SendInterval   time.Duration `toml:"send_interval"`
	SendTimeout    time.Duration `toml:"send_timeout"`
	RepeatInterval time.Duration `toml:"repeat_interval"`
}

func LoadConfig(configPath string, resolverFunc util.MXResolverFunc) (*Config, error) {
	slog.Info("start create config")

	cfg := &Config{}

	if configPath == "" {
		panic(fmt.Errorf("config file path is required"))
	}

	bytes, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %s", err)
	}

	if err := toml.Unmarshal(bytes, cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config: %s", err)
	}

	if cfg.ScrapeInterval.Seconds() == 0 {
		cfg.ScrapeInterval = time.Minute
	}

	if cfg.MgrCtlPath == "" {
		cfg.MgrCtlPath = isp.MGR_CTL_PATH_DEFAULT
	}

	if cfg.SMTP.Password == "" || cfg.SMTP.Port == "" || cfg.SMTP.Username == "" {
		return nil, fmt.Errorf("check SMTP settings")
	}

	if cfg.SMTP.Host == "" {
		host, err := resolverFunc(cfg.SMTP.Username)
		if err != nil {
			return nil, fmt.Errorf("failed fill username and host: %w", err)
		}

		cfg.SMTP.Host = host
	}

	if cfg.EMail.From == "" || len(cfg.EMail.To) == 0 || cfg.EMail.Subject == "" {
		return nil, fmt.Errorf("check mail settings")
	}

	if cfg.SendInterval.Seconds() == 0 {
		cfg.SendInterval = time.Minute * 1
	}

	if cfg.SiteRetentionInterval.Seconds() == 0 {
		cfg.SiteRetentionInterval = cfg.ScrapeInterval * 4
	}

	if cfg.RepeatInterval.Minutes() == 0 {
		cfg.RepeatInterval = time.Hour * 4
	}

	if cfg.SendTimeout.Seconds() == 0 {
		cfg.SendTimeout = time.Second * 2
	}

	if cfg.SendInterval.Seconds()-cfg.SendTimeout.Seconds() <= 2 {
		return nil, fmt.Errorf("interval must be greater than timeout by more than 2 seconds")
	}

	return cfg, nil
}
