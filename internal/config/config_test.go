package config

import (
	"flag"
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
recipient = "kias@gendalf.ru"

[smtp]
username = "test@test.tu"
password = "hello-world"
host = "mail.yandex.ru"
port = 465
from = "test@test.tu"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	oldArgs := os.Args
	oldCommandLine := flag.CommandLine
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
	}()

	os.Args = []string{"cmd", "-config", configPath}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("произошла ошибка при создании конфигурации: %v", err)
	}

	assert.Equal(t, "/usr/local/mgr5/sbin/mgrctl", cfg.MgrCtlPath)
	assert.Equal(t, "kias@gendalf.ru", cfg.Recipient)
	assert.Equal(t, "1m0s", cfg.ScrapeInterval.String())
	assert.Equal(t, "test@test.tu", cfg.SMTP.From)
	assert.Equal(t, "test@test.tu", cfg.SMTP.Username)
	assert.Equal(t, "hello-world", cfg.SMTP.Password)
	assert.Equal(t, "mail.yandex.ru", cfg.SMTP.Host)
	assert.Equal(t, 465, cfg.SMTP.Port)
}

func TestLoadConfig_Defaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	configContent := `
recipient = "kias@gendalf.ru"

[smtp]
username = "test@test.tu"
password = "hello-world"
host = "mail.yandex.ru"
port = 465
from = "test@test.tu"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	oldArgs := os.Args
	oldCommandLine := flag.CommandLine
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
	}()

	os.Args = []string{"cmd", "-config", configPath}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	cfg, err := LoadConfig()

	assert.Nil(t, err)
	assert.Equal(t, "/usr/local/mgr5/sbin/mgrctl", cfg.MgrCtlPath)
	assert.Equal(t, "kias@gendalf.ru", cfg.Recipient)
	assert.Equal(t, "1m0s", cfg.ScrapeInterval.String())
	assert.Equal(t, "test@test.tu", cfg.SMTP.From)
	assert.Equal(t, "test@test.tu", cfg.SMTP.Username)
	assert.Equal(t, "hello-world", cfg.SMTP.Password)
	assert.Equal(t, "mail.yandex.ru", cfg.SMTP.Host)
	assert.Equal(t, 465, cfg.SMTP.Port)
}

func TestLoadConfig_EmptyRecipient(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	configContent := `
[smtp]
username = "test@test.tu"
password = "hello-world"
host = "mail.yandex.ru"
port = 465
from = "test@test.tu"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	oldArgs := os.Args
	oldCommandLine := flag.CommandLine
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
	}()

	os.Args = []string{"cmd", "-config", configPath}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	_, err = LoadConfig()
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

	oldArgs := os.Args
	oldCommandLine := flag.CommandLine
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
	}()

	os.Args = []string{"cmd", "-config", configPath}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	_, err = LoadConfig()
	assert.Error(t, err)
}

func TestLoadConfig_EmptySMTPTimeoutAndInterval(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	configContent := `
recipient = "test@test.ru"

[smtp]
username = "test@test.tu"
password = "hello-world"
host = "mail.yandex.ru"
port = 465
from = "test@test.tu"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	oldArgs := os.Args
	oldCommandLine := flag.CommandLine
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
	}()

	os.Args = []string{"cmd", "-config", configPath}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("ошибка чтения конфигурации %v", err)
	}

	assert.Equal(t, "1m0s", cfg.SMTP.SendInterval.String())
	assert.Equal(t, "2s", cfg.SMTP.SendTimeout.String())
}
