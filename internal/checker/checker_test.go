package checker

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/smtp"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kias-hack/isp-site-checker/internal/config"
	"github.com/kias-hack/isp-site-checker/internal/isp"
	"github.com/stretchr/testify/assert"
)

func TestNewChecker(t *testing.T) {
	chk := NewChecker(&config.Config{
		ScrapeInterval: time.Second,
	})

	assert.NotNil(t, chk.cancel)
	assert.NotNil(t, chk.config)
	assert.NotNil(t, chk.ctx)
	assert.NotNil(t, chk.resultPipe)
	assert.NotNil(t, chk.wg)
	assert.False(t, chk.work)
}

func TestCheckerStart(t *testing.T) {
	chk := NewChecker(&config.Config{
		ScrapeInterval: time.Second,
	})

	startGoroutine := runtime.NumGoroutine()

	if err := chk.Start(); err != nil {
		t.Fatalf("ошибка при старте сервиса %v", err)
	}

	assert.Equal(t, 2, runtime.NumGoroutine()-startGoroutine)

	assert.True(t, chk.work)

	chk.Stop(t.Context())
}

func TestCheckerRetryStart(t *testing.T) {
	chk := NewChecker(&config.Config{
		ScrapeInterval: time.Second,
	})
	defer chk.Stop(t.Context())

	chk.Start()

	assert.Error(t, chk.Start())
}

func TestCheckerStop(t *testing.T) {
	chk := NewChecker(&config.Config{
		ScrapeInterval: time.Second,
	})

	startGoroutine := runtime.NumGoroutine()
	chk.Start()

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()

	assert.Nil(t, chk.Stop(ctx), "ошибка при заврешении сервиса")
	assert.Zero(t, runtime.NumGoroutine()-startGoroutine, "не все горутины завершились после остановки сервиса")
}

func TestCheckerStopWithTimeout(t *testing.T) {
	chk := NewChecker(&config.Config{
		ScrapeInterval: time.Second,
	})
	chk.Start()

	chk.wg.Add(1)
	go func() {
		time.Sleep(1 * time.Second)
		chk.wg.Done()
	}()

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()

	err := chk.Stop(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "контекст завершился")
}

func TestCheckerNotification(t *testing.T) {
	chk := NewChecker(&config.Config{
		ScrapeInterval: time.Second,
	})
	chk.Start()
	defer chk.Stop(t.Context())
	wg := &sync.WaitGroup{}
	defer func() {
		sendMail = smtp.SendMail
	}()

	testCases := []struct {
		name     string
		dataFunc func() (CheckInfo, []byte)
	}{
		{
			name:     "результат с ошибкой",
			dataFunc: getGetTestDataForNotifyWithError,
		},
		{
			name:     "результат с неверным статусом",
			dataFunc: getGetTestDataForNotifyWithStatus,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, expected := testCase.dataFunc()
			wg.Add(1)

			//mock sendMail
			sendMail = func(_ string, _ smtp.Auth, _ string, _ []string, msg []byte) error {
				assert.Equal(t, expected, msg)
				wg.Done()
				return nil
			}

			// отправляем результат
			chk.resultPipe <- result

			// дожидаемся что результат обработается
			exit := make(chan struct{})
			go func() {
				wg.Wait()
				close(exit)
			}()

			timeout := time.NewTimer(1 * time.Second)
			defer timeout.Stop()

			select {
			case <-exit:
			case <-timeout.C:
			}
		})
	}
}

func TestCheckerNotificationValidStatus(t *testing.T) {
	chk := NewChecker(&config.Config{
		ScrapeInterval: time.Second,
	})
	chk.Start()
	defer func() {
		sendMail = smtp.SendMail
	}()
	defer chk.Stop(t.Context())

	//mock sendMail
	sendMail = func(_ string, _ smtp.Auth, _ string, _ []string, msg []byte) error {
		t.Fatalf("было отправлено уведомление, не должно быть так")
		return nil
	}

	result := CheckInfo{
		Domain:     "example.com",
		Owner:      "root",
		Timestamp:  time.Now(),
		StatusCode: http.StatusForbidden,
	}

	// отправляем результат
	chk.resultPipe <- result

	// дожидаемся что результат обработается
	runtime.Gosched()
	time.Sleep(50 * time.Millisecond)
}

func getGetTestDataForNotifyWithError() (CheckInfo, []byte) {
	result := CheckInfo{
		Domain:    "example.com",
		Owner:     "root",
		Timestamp: time.Now(),
		Err:       fmt.Errorf("some error"),
	}

	expected := strings.Builder{}
	expected.WriteString("Проверка домена выявила проблему\n")
	expected.WriteString(fmt.Sprintf("Домен: %s\n", result.Domain))
	expected.WriteString(fmt.Sprintf("Владелец: %s\n", result.Owner))
	expected.WriteString(fmt.Sprintf("Время: %s\n", result.Timestamp))
	expected.WriteString("\n")
	expected.WriteString(fmt.Sprintf("Произошла ошибка - %s", result.Err.Error()))

	return result, []byte(expected.String())
}

func getGetTestDataForNotifyWithStatus() (CheckInfo, []byte) {
	result := CheckInfo{
		Domain:     "example.com",
		Owner:      "root",
		Timestamp:  time.Now(),
		StatusCode: http.StatusOK,
	}

	expected := strings.Builder{}
	expected.WriteString("Проверка домена выявила проблему\n")
	expected.WriteString(fmt.Sprintf("Домен: %s\n", result.Domain))
	expected.WriteString(fmt.Sprintf("Владелец: %s\n", result.Owner))
	expected.WriteString(fmt.Sprintf("Время: %s\n", result.Timestamp))
	expected.WriteString("\n")
	expected.WriteString(fmt.Sprintf("Код ответа - %d", result.StatusCode))

	return result, []byte(expected.String())
}

func TestCheckerCheckSendRequest(t *testing.T) {
	chk := NewChecker(&config.Config{
		ScrapeInterval: time.Second,
	})
	defer chk.Stop(t.Context())

	wg := &sync.WaitGroup{}

	wg.Add(2) // 1 домен и 1 поддомен
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info("call")

		w.WriteHeader(http.StatusOK)
		wg.Done()
	}))
	defer server.Close()

	serverURL, _ := url.Parse(server.URL)

	getDomains = func(_ string) ([]*isp.WebDomain, error) {
		return []*isp.WebDomain{
			{
				Id:      1,
				Name:    "example.com",
				Owner:   "root",
				Docroot: "/",
				Active:  true,
				IPAddr:  serverURL.Hostname(),
				Port:    serverURL.Port(),
			},
			{
				Id:      1,
				Name:    "123example.com",
				Owner:   "root",
				Docroot: "/",
				Active:  false,
				IPAddr:  serverURL.Hostname(),
				Port:    serverURL.Port(),
			},
		}, nil
	}

	readDir = func(name string) ([]os.DirEntry, error) {
		return []os.DirEntry{
			dirEntry{
				name:  "www.example.com",
				isDir: true,
			},
		}, nil
	}

	sendMail = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		return nil
	}

	defer func() {
		sendMail = smtp.SendMail
		readDir = os.ReadDir
		getDomains = isp.GetWebDomain
	}()

	chk.Start()

	exit := make(chan struct{})

	go func() {
		wg.Wait()
		close(exit)
	}()

	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	select {
	case <-timer.C:
		t.Fatalf("таймаут, задачи не зевершены")
	case <-exit:
	}

}
