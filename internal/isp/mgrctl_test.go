package isp

import (
	"fmt"
	"os/exec"
	"testing"
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
			domains, err := GetWebDomain("mgrctl")
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
