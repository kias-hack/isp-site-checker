package checker

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
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
	gomock "go.uber.org/mock/gomock"
)

func TestNewChecker(t *testing.T) {
	ctrl, sender := newMockSender(t)
	defer ctrl.Finish()

	chk := NewChecker(&config.Config{
		ScrapeInterval: time.Second,
	}, sender)

	assert.NotNil(t, chk.cancel)
	assert.NotNil(t, chk.config)
	assert.NotNil(t, chk.ctx)
	assert.NotNil(t, chk.resultPipe)
	assert.NotNil(t, chk.wg)
	assert.False(t, chk.work)
}

func TestCheckerStart(t *testing.T) {
	ctrl, sender := newMockSender(t)
	defer ctrl.Finish()

	chk := NewChecker(&config.Config{
		ScrapeInterval: time.Second,
	}, sender)

	startGoroutine := runtime.NumGoroutine()

	if err := chk.Start(); err != nil {
		t.Fatalf("ошибка при старте сервиса %v", err)
	}

	assert.Equal(t, 3, runtime.NumGoroutine()-startGoroutine)

	assert.True(t, chk.work)

	chk.Stop(t.Context())
}

func TestCheckerRetryStart(t *testing.T) {
	ctrl, sender := newMockSender(t)
	defer ctrl.Finish()

	chk := NewChecker(&config.Config{
		ScrapeInterval: time.Second,
	}, sender)
	defer chk.Stop(t.Context())

	chk.Start()

	assert.Error(t, chk.Start())
}

func TestCheckerStop(t *testing.T) {
	ctrl, sender := newMockSender(t)
	defer ctrl.Finish()

	chk := NewChecker(&config.Config{
		ScrapeInterval: time.Second,
	}, sender)

	startGoroutine := runtime.NumGoroutine()
	chk.Start()

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()

	assert.Nil(t, chk.Stop(ctx), "ошибка при заврешении сервиса")
	assert.Zero(t, runtime.NumGoroutine()-startGoroutine, "не все горутины завершились после остановки сервиса")
}

func TestCheckerStopWithTimeout(t *testing.T) {
	ctrl, sender := newMockSender(t)
	defer ctrl.Finish()

	chk := NewChecker(&config.Config{
		ScrapeInterval: time.Second,
	}, sender)
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
	wg := &sync.WaitGroup{}

	testCases := []struct {
		name     string
		dataFunc func() (CheckInfo, string)
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

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockSender := NewMockNotificationSender(ctrl)
			mockSender.EXPECT().Error(gomock.Eq(result.Domain), expected).Times(1).Do(func(_, _ string) {
				wg.Done()
			})
			mockSender.EXPECT().Send(gomock.Any()).Return(nil).AnyTimes()

			wg.Add(1)

			chk := NewChecker(&config.Config{
				ScrapeInterval: time.Second,
			}, mockSender)
			chk.Start()
			defer chk.Stop(t.Context())

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
	ctrl, sender := newMockSender(t)
	defer ctrl.Finish()

	chk := NewChecker(&config.Config{
		ScrapeInterval: time.Second,
	}, sender)

	chk.Start()
	defer chk.Stop(t.Context())

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

func getGetTestDataForNotifyWithError() (CheckInfo, string) {
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

	return result, expected.String()
}

func getGetTestDataForNotifyWithStatus() (CheckInfo, string) {
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

	return result, expected.String()
}

func TestCheckerCheckSendRequest(t *testing.T) {
	ctrl, sender := newMockSender(t)
	defer ctrl.Finish()

	chk := NewChecker(&config.Config{
		ScrapeInterval: time.Second,
	}, sender)
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

	defer func() {
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

func TestCheckerCheckThatSendResultCalls(t *testing.T) {
	ctrl := gomock.NewController(t)
	sender := NewMockNotificationSender(ctrl)
	defer ctrl.Finish()

	wg := &sync.WaitGroup{}

	wg.Add(2)

	sender.EXPECT().Send(gomock.Any()).Times(1).DoAndReturn(func(_ context.Context) error {
		wg.Done()
		return nil
	})
	sender.EXPECT().Error(gomock.Any(), gomock.Any()).Times(1).Do(func(_, _ string) {
		wg.Done()
	})

	chk := NewChecker(&config.Config{
		ScrapeInterval: time.Second,
	}, sender)
	defer chk.Stop(t.Context())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
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
		}, nil
	}

	readDir = func(name string) ([]os.DirEntry, error) {
		return []os.DirEntry{}, nil
	}

	defer func() {
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

// newMockSender создаёт мок с дефолтными ожиданиями (Error/Success — любое кол-во, Send — nil).
func newMockSender(t *testing.T) (*gomock.Controller, *MockNotificationSender) {
	ctrl := gomock.NewController(t)
	mock := NewMockNotificationSender(ctrl)
	mock.EXPECT().Error(gomock.Any(), gomock.Any()).AnyTimes()
	mock.EXPECT().Success(gomock.Any()).AnyTimes()
	mock.EXPECT().Send(gomock.Any()).Return(nil).AnyTimes()
	return ctrl, mock
}
