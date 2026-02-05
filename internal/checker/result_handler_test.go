package checker

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/kias-hack/isp-site-checker/internal/notify"
	"go.uber.org/mock/gomock"
)

func TestResultHandlerLifecycle(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	resultPipe := make(chan *Task)
	wg := &sync.WaitGroup{}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notifierStub := notify.NewMockNotifier(ctrl)
	notifierStub.EXPECT().Fail(gomock.Any(), gomock.Any()).Times(0)
	notifierStub.EXPECT().Success(gomock.Any(), gomock.Any()).Times(0)

	wg.Add(1)
	go resultHandler(ctx, wg, resultPipe, notifierStub)

	cancel()

	exit := make(chan struct{})
	go func() {
		wg.Wait()
		close(exit)
	}()

	timeoutCtx, cancel := context.WithTimeout(t.Context(), 1*time.Second)
	defer cancel()
	select {
	case <-timeoutCtx.Done():
		t.Fatal("таймаут завершения")
	case <-exit:
	}
}

func TestResultCases(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	resultPipe := make(chan *Task)
	wg := &sync.WaitGroup{}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notifierMock := notify.NewMockNotifier(ctrl)

	wg.Add(1)
	go resultHandler(ctx, wg, resultPipe, notifierMock)

	testCases := []struct {
		name           string
		expectedMethod string
		task           *Task
		expectedText   string
	}{
		{
			name:           "success",
			expectedMethod: "Success",
			task: &Task{
				DomainId:   1,
				Site:       "example.com",
				DomainName: "example.com",
				Owner:      "root",
				Result: struct {
					StatusCode int
					Err        error
					Timestamp  time.Time
				}{
					StatusCode: http.StatusForbidden,
				},
			},
			expectedText: "Сайт example.com закрыт - 403\r\nВладелец - root",
		},
		{
			name:           "200 - ok",
			expectedMethod: "Fail",
			task: &Task{
				DomainId:   1,
				Site:       "example.com",
				DomainName: "example.com",
				Owner:      "root",
				Result: struct {
					StatusCode int
					Err        error
					Timestamp  time.Time
				}{
					StatusCode: http.StatusOK,
					Timestamp:  time.Date(2026, 2, 6, 1, 1, 1, 1, time.UTC),
				},
			},
			expectedText: "Проверка домена выявила проблему\nСайт: example.com\nВладелец: root\nВремя: 2026-02-06 01:01:01.000000001 +0000 UTC\nКод ответа: 200",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.expectedMethod == "Fail" {
				notifierMock.EXPECT().Fail(testCase.task.Site, testCase.expectedText).Times(1)
			} else {
				notifierMock.EXPECT().Fail(gomock.Any(), gomock.Any()).Times(0)
			}

			if testCase.expectedMethod == "Success" {
				notifierMock.EXPECT().Success(testCase.task.Site, testCase.expectedText).Times(1)
			} else {
				notifierMock.EXPECT().Success(gomock.Any(), gomock.Any()).Times(0)
			}

			resultPipe <- testCase.task
		})
	}

	cancel()

	exit := make(chan struct{})
	go func() {
		wg.Wait()
		close(exit)
	}()

	timeoutCtx, cancel := context.WithTimeout(t.Context(), 1*time.Second)
	defer cancel()
	select {
	case <-timeoutCtx.Done():
		t.Fatal("таймаут завершения")
	case <-exit:
	}
}
