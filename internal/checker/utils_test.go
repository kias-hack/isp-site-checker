package checker

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
