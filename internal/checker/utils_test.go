package checker

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

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

func TestCreateClientDialContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	defer server.Close()

	serverURL, _ := url.Parse(server.URL)
	client := createClient(serverURL.Hostname(), serverURL.Port())

	resp, err := client.Get("http://yandex.ru/")
	if err != nil {
		t.Fatalf("ошибка запроса к тестовому серверу %v", err)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("произошла ошибка при получении данных ответа %v", err)
	}

	assert.Equal(t, []byte("OK"), data)
}
