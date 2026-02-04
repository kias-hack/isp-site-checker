package checker

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/kias-hack/isp-site-checker/internal/isp"
)

func TestLifecycle(t *testing.T) {
	wg := &sync.WaitGroup{}

	ctx, cancel := context.WithCancel(t.Context())

	wg.Add(1)
	go sсheduler(ctx, wg, make(<-chan struct{}), make(chan<- *Task), func() ([]*isp.WebDomain, error) {
		return nil, nil
	})

	runtime.Gosched()

	cancel()

	exit := make(chan struct{})
	go func() {
		wg.Wait()
		close(exit)
	}()

	time.Sleep(10 * time.Millisecond)
	if _, ok := <-exit; ok {
		t.Fatal("планировщик не завершил свою работу после отсановки контекста")
	}
}

func TestGetWebdomainsFailure(t *testing.T) {
	wg := &sync.WaitGroup{}

	ctx, cancel := context.WithCancel(t.Context())

	wg.Add(1)
	ticker := make(chan struct{})
	taskPipe := make(chan *Task)
	defer func() {
		close(ticker)
		close(taskPipe)
	}()

	go sсheduler(ctx, wg, ticker, taskPipe, func() ([]*isp.WebDomain, error) {
		return []*isp.WebDomain{
			{Sites: []string{"example.com"}},
		}, fmt.Errorf("test error")
	})
	defer cancel()

	ticker <- struct{}{}

	time.Sleep(10 * time.Millisecond)

	select {
	case task := <-taskPipe:
		if task != nil {
			t.Fatal("ошибка, планировщик отправил задачу", "task", task)
		}
	default:
		t.Log("задача отсутствует")
	}

	cancel()
	wg.Wait()
}
