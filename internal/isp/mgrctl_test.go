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
	}{
		{
			name:       "domain with digits",
			id:         1,
			domainName: "123owner.owner-oner.er",
			owner:      "owner",
			docroot:    "/var/www/owner/data/www/123owner.owner-oner.er",
			ipAddr:     "127.0.0.1",
		},
		{
			name:       "domain without digits",
			id:         2,
			domainName: "example.com",
			owner:      "owner1",
			docroot:    "/var/www/owner1/data/www/example.com",
			ipAddr:     "192.23.3.3",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Mock mgrctl command
			execCommand = func(path string, args ...string) *exec.Cmd {
				result := fmt.Sprintf(
					webDomainLineTemplate,
					tc.id,
					tc.domainName,
					tc.owner,
					tc.docroot,
					"on",
					tc.ipAddr,
				)

				return exec.Command("echo", result)
			}

			defer func() {
				execCommand = exec.Command
			}()

			// Run function
			domains, err := GetWebDomains("mgrctl")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check domain count
			if len(domains) != 1 {
				t.Fatalf("got %d domains, expected 1", len(domains))
			}

			domain := domains[0]

			// Check all fields
			if domain.Id != tc.id {
				t.Errorf("Id: got %d, expected %d", domain.Id, tc.id)
			}

			if domain.Name != tc.domainName {
				t.Errorf("Name: got %q, expected %q", domain.Name, tc.domainName)
			}

			if domain.Owner != tc.owner {
				t.Errorf("Owner: got %q, expected %q", domain.Owner, tc.owner)
			}

			if domain.Docroot != tc.docroot {
				t.Errorf("Docroot: got %q, expected %q", domain.Docroot, tc.docroot)
			}

			if domain.IPAddr != tc.ipAddr {
				t.Errorf("IPAddr: got %q, expected %q", domain.IPAddr, tc.ipAddr)
			}
		})
	}
}

func TestGetWebDomainThatNotActiveWillBeIgnored(t *testing.T) {
	// Mock mgrctl command
	execCommand = func(path string, args ...string) *exec.Cmd {
		result := fmt.Sprintf(
			webDomainLineTemplate,
			1,
			"example.com",
			"root",
			"/var/www/test",
			"off",
			"127.0.0.0.1",
		)

		return exec.Command("echo", result)
	}

	defer func() {
		execCommand = exec.Command
	}()

	// Run function
	domains, err := GetWebDomains("mgrctl")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check domain count
	if len(domains) != 0 {
		t.Fatalf("inactive domain was incorrectly returned")
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
			name: "search with multiple files",
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
			name: "search with main domain present",
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
			name: "main domain and backup subdomain",
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
			name: "main domain and multiple subdomains",
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
			name: "main domain and subdomains with one being a file",
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
			name: "no matching directories",
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
			name:           "empty directory",
			entries:        []os.DirEntry{},
			expectedResult: []string{"other-domain.ru"},
			domain:         "other-domain.ru",
		},
		{
			name: "similar name but not subdomain",
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
			name: "similar name but not subdomain (duplicate)",
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
