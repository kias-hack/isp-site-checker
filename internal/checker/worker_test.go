package checker

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestWorkerLifecycle(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	wg := &sync.WaitGroup{}

	wg.Add(1)
	go worker(ctx, wg, make(<-chan *Task), make(chan<- *Task), 0)

	runtime.Gosched()

	exit := make(chan struct{})
	cancel()

	go func() {
		wg.Wait()
		close(exit)
	}()

	checkCtx, cancel := context.WithTimeout(t.Context(), 1*time.Second)
	defer cancel()

	select {
	case <-checkCtx.Done():
		t.Fatal("таймаут при заврешении воркера")
	case <-exit:
	}
}

// TODO обработка ошибки при запросе к домену
// TODO обработка неверного статуса
// TODO обработка верного статуса
// TODO проверить на утечку на множестве запросов
// TODO завершение по контексту при отправке запроса
