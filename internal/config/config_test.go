package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	configContent := `
scrape_interval = "60s"
mgrctl_path = "/usr/local/mgr5/sbin/mgrctl"

[smtp]
email = "test@test.tu"
password = "hello-world"
port = "465"

[email]
to = ["test@example.com"]
subject = "subject"
from = "test@test.tu"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(configPath, func(email string) (host string, err error) {
		return "mail.test.tu", nil
	})
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	assert.Equal(t, "/usr/local/mgr5/sbin/mgrctl", cfg.MgrCtlPath)
	assert.Equal(t, "1m0s", cfg.ScrapeInterval.String())
	assert.Equal(t, "test@test.tu", cfg.EMail.From)
	assert.Equal(t, []string{"test@example.com"}, cfg.EMail.To)
	assert.Equal(t, "subject", cfg.EMail.Subject)
	assert.Equal(t, "test@test.tu", cfg.SMTP.Username)
	assert.Equal(t, "hello-world", cfg.SMTP.Password)
	assert.Equal(t, "mail.test.tu", cfg.SMTP.Host)
	assert.Equal(t, "465", cfg.SMTP.Port)
}

func TestLoadConfig_Defaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	configContent := `
[smtp]
email = "test@test.tu"
password = "hello-world"
port = "465"

[email]
to = ["test@example.com"]
subject = "subject"
from = "test@test.tu"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(configPath, func(email string) (host string, err error) {
		return "mail.test.tu", nil
	})

	assert.Nil(t, err)
	assert.Equal(t, "/usr/local/mgr5/sbin/mgrctl", cfg.MgrCtlPath)
	assert.Equal(t, "1m0s", cfg.ScrapeInterval.String())
	assert.Equal(t, "test@test.tu", cfg.EMail.From)
	assert.Equal(t, []string{"test@example.com"}, cfg.EMail.To)
	assert.Equal(t, "subject", cfg.EMail.Subject)
	assert.Equal(t, "test@test.tu", cfg.SMTP.Username)
	assert.Equal(t, "hello-world", cfg.SMTP.Password)
	assert.Equal(t, "mail.test.tu", cfg.SMTP.Host)
	assert.Equal(t, "4m0s", cfg.SiteRetentionInterval.String())
	assert.Equal(t, "465", cfg.SMTP.Port)
}

func TestLoadConfig_AlternativeSMTPHost(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	configContent := `
[smtp]
email = "test@test.tu"
password = "hello-world"
host = "smtp.somehost.tu"
port = "465"

[email]
to = ["test@example.com"]
subject = "subject"
from = "test@test.tu"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(configPath, func(email string) (host string, err error) {
		return "mail.test.tu", nil
	})

	assert.Nil(t, err)
	assert.Equal(t, "/usr/local/mgr5/sbin/mgrctl", cfg.MgrCtlPath)
	assert.Equal(t, "1m0s", cfg.ScrapeInterval.String())
	assert.Equal(t, "test@test.tu", cfg.EMail.From)
	assert.Equal(t, []string{"test@example.com"}, cfg.EMail.To)
	assert.Equal(t, "subject", cfg.EMail.Subject)
	assert.Equal(t, "test@test.tu", cfg.SMTP.Username)
	assert.Equal(t, "hello-world", cfg.SMTP.Password)
	assert.Equal(t, "smtp.somehost.tu", cfg.SMTP.Host)
	assert.Equal(t, "465", cfg.SMTP.Port)
}

func TestLoadConfig_EmptyRecipient(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	configContent := `
[smtp]
email = "test@test.tu"
password = "hello-world"
port = "465"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	_, err = LoadConfig(configPath, func(email string) (host string, err error) {
		return "mail.test.tu", nil
	})
	assert.Error(t, err)
}

func TestLoadConfig_EmptySMTP(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	configContent := `
recipient = "test@test.ru"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	_, err = LoadConfig(configPath, func(email string) (host string, err error) {
		return "mail.test.tu", nil
	})
	assert.Error(t, err)
}

func TestLoadConfig_EmptySMTPTimeoutAndInterval(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	configContent := `
[smtp]
email = "test@test.tu"
password = "hello-world"
port = "465"

[email]
to = ["test@example.com"]
subject = "subject"
from = "test@test.tu"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(configPath, func(email string) (host string, err error) {
		return "mail.test.tu", nil
	})
	if err != nil {
		t.Fatalf("config load error: %v", err)
	}

	assert.Equal(t, "1m0s", cfg.SendInterval.String())
	assert.Equal(t, "2s", cfg.SendTimeout.String())
}
