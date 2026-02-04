package isp

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	webDomainLineTemplate = "id=%d name=%s owner=%s docroot=%s php=Path to PHP: /opt/php82/bin/php. php_mode=php_mode_mod php_version=8.2.29 (alt) handler=PHP Apache 8.2.29 (alt) active=%s analyzer=off ipaddr=%s webscript_status= database=db_not_assigned ssl_not_used="
)

func TestGetWebDomain(t *testing.T) {
	testCases := []struct {
		name       string
		id         int
		domainName string
		owner      string
		docroot    string
		ipAddr     string
		active     bool
	}{
		{
			name:       "активный домен с цифрами",
			id:         1,
			domainName: "123owner.owner-oner.er",
			owner:      "owner",
			docroot:    "/var/www/owner/data/www/123owner.owner-oner.er",
			ipAddr:     "127.0.0.1",
			active:     true,
		},
		{
			name:       "неактивный домен",
			id:         2,
			domainName: "123owner.owner1-oner.er",
			owner:      "owner1",
			docroot:    "/var/www/owner1/data/www/123owner.owner1-oner.er",
			ipAddr:     "192.23.3.3",
			active:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Мокаем команду mgrctl
			execCommand = func(path string, args ...string) *exec.Cmd {
				activeStr := "off"
				if tc.active {
					activeStr = "on"
				}

				result := fmt.Sprintf(
					webDomainLineTemplate,
					tc.id,
					tc.domainName,
					tc.owner,
					tc.docroot,
					activeStr,
					tc.ipAddr,
				)

				return exec.Command("echo", result)
			}

			defer func() {
				execCommand = exec.Command
			}()

			// Выполняем функцию
			domains, err := GetWebDomains("mgrctl")
			if err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}

			// Проверяем количество доменов
			if len(domains) != 1 {
				t.Fatalf("получено доменов: %d, ожидается: 1", len(domains))
			}

			domain := domains[0]

			// Проверяем все поля
			if domain.Id != tc.id {
				t.Errorf("Id: получено %d, ожидается %d", domain.Id, tc.id)
			}

			if domain.Name != tc.domainName {
				t.Errorf("Name: получено %q, ожидается %q", domain.Name, tc.domainName)
			}

			if domain.Owner != tc.owner {
				t.Errorf("Owner: получено %q, ожидается %q", domain.Owner, tc.owner)
			}

			if domain.Docroot != tc.docroot {
				t.Errorf("Docroot: получено %q, ожидается %q", domain.Docroot, tc.docroot)
			}

			if domain.IPAddr != tc.ipAddr {
				t.Errorf("IPAddr: получено %q, ожидается %q", domain.IPAddr, tc.ipAddr)
			}

			if domain.Active != tc.active {
				t.Errorf("Active: получено %v, ожидается %v", domain.Active, tc.active)
			}
		})
	}
}

type dirEntry struct {
	name  string
	isDir bool
}

func (d dirEntry) Name() string               { return d.name }
func (d dirEntry) IsDir() bool                { return d.isDir }
func (d dirEntry) Type() os.FileMode          { return 0 }
func (d dirEntry) Info() (os.FileInfo, error) { return nil, nil }

func TestFindSubdomainWithDirs(t *testing.T) {
	testCases := []struct {
		name           string
		entries        []os.DirEntry
		expectedResult []string
		domain         string
	}{
		{
			name: "Поиск с несколькими файлами",
			entries: []os.DirEntry{
				dirEntry{
					name:  "certs",
					isDir: true,
				},
				dirEntry{
					name:  "db.sql",
					isDir: false,
				},
				dirEntry{
					name:  "db.sql",
					isDir: false,
				},
				dirEntry{
					name:  "post.localhost.ru",
					isDir: true,
				},
			},
			expectedResult: []string{"localhost.ru", "post.localhost.ru"},
			domain:         "localhost.ru",
		},
		{
			name: "Поиск с несколькими файлами и при наличии основного домена",
			entries: []os.DirEntry{
				dirEntry{
					name:  "localhost.ru",
					isDir: true,
				},
				dirEntry{
					name:  "post.localhost.ru",
					isDir: true,
				},
			},
			expectedResult: []string{"localhost.ru", "post.localhost.ru"},
			domain:         "localhost.ru",
		},
		{
			name: "в наличии основной домен и поддомен но забэкапированный",
			entries: []os.DirEntry{
				dirEntry{
					name:  "localhost.ru",
					isDir: true,
				},
				dirEntry{
					name:  "post.localhost.ru.backup",
					isDir: true,
				},
			},
			expectedResult: []string{"localhost.ru"},
			domain:         "localhost.ru",
		},
		{
			name: "в наличии основной домен и несколько поддоменов",
			entries: []os.DirEntry{
				dirEntry{
					name:  "localhost.ru",
					isDir: true,
				},
				dirEntry{
					name:  "post.localhost.ru",
					isDir: true,
				},
				dirEntry{
					name:  "new.localhost.ru",
					isDir: true,
				},
			},
			expectedResult: []string{"localhost.ru", "post.localhost.ru", "new.localhost.ru"},
			domain:         "localhost.ru",
		},
		{
			name: "в наличии основной домен и несколько поддоменов, но один и поддоменов файлом является",
			entries: []os.DirEntry{
				dirEntry{
					name:  "localhost.ru",
					isDir: true,
				},
				dirEntry{
					name:  "post.localhost.ru",
					isDir: true,
				},
				dirEntry{
					name:  "new.localhost.ru",
					isDir: false,
				},
			},
			expectedResult: []string{"localhost.ru", "post.localhost.ru"},
			domain:         "localhost.ru",
		},
		{
			name: "нет никаких директорий",
			entries: []os.DirEntry{
				dirEntry{
					name:  "localhost.ru",
					isDir: true,
				},
				dirEntry{
					name:  "post.localhost.ru",
					isDir: true,
				},
				dirEntry{
					name:  "new.localhost.ru",
					isDir: false,
				},
			},
			expectedResult: []string{"other-domain.ru"},
			domain:         "other-domain.ru",
		},
		{
			name:           "пустая директория",
			entries:        []os.DirEntry{},
			expectedResult: []string{"other-domain.ru"},
			domain:         "other-domain.ru",
		},
		{
			name: "есть домен с похожим именем, но это не поддомен",
			entries: []os.DirEntry{
				dirEntry{
					name:  "oooother-domain.ru",
					isDir: true,
				},
			},
			expectedResult: []string{"other-domain.ru"},
			domain:         "other-domain.ru",
		},
		{
			name: "есть домен с похожим именем, но это не поддомен",
			entries: []os.DirEntry{
				dirEntry{
					name:  "test.oooother-domain.ru",
					isDir: true,
				},
			},
			expectedResult: []string{"other-domain.ru"},
			domain:         "other-domain.ru",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			readDir = func(_ string) ([]os.DirEntry, error) {
				return testCase.entries, nil
			}

			defer func() {
				readDir = os.ReadDir
			}()

			domains := findSubdomain("root", testCase.domain)

			assert.Equal(t, testCase.expectedResult, domains)
		})
	}
}

func TestFindSubdomainWhenDirNotExist(t *testing.T) {
	readDir = func(name string) ([]os.DirEntry, error) {
		return nil, os.ErrNotExist
	}
	domain := "domain.ru"

	expected := []string{domain}

	result := findSubdomain("root", domain)

	assert.Equal(t, expected, result)
}

func TestFindSubdomainWhenUnknownError(t *testing.T) {
	readDir = func(name string) ([]os.DirEntry, error) {
		return nil, os.ErrPermission
	}

	defer func() {
		readDir = os.ReadDir
	}()

	domain := "domain.ru"

	assert.Panics(t, func() {
		_ = findSubdomain("root", domain)
	})
}
