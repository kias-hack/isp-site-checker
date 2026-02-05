package checker

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

type RoundTripperWithMock struct {
	callChan chan struct{}
}

func (r *RoundTripperWithMock) RoundTrip(*http.Request) (*http.Response, error) {
	r.callChan <- struct{}{}
	<-r.callChan

	return nil, fmt.Errorf("some error")
}

func TestWorkerLifecycle(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	wg := &sync.WaitGroup{}

	wg.Add(1)
	go worker(ctx, wg, make(<-chan *Task), make(chan<- *Task), 0)

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

func TestWorkerStopByContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	wg := &sync.WaitGroup{}
	taskCh := make(chan *Task)
	resultPipe := make(chan *Task, 1)
	defer func() {
		close(taskCh)
		close(resultPipe)
	}()

	wg.Add(1)
	go worker(ctx, wg, taskCh, resultPipe, 0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	serverUrl, _ := url.Parse(server.URL)

	expectedTask := &Task{
		Site:     site,
		DomainId: 1,
		Owner:    owner,
		Connection: struct {
			Addr string
			Port string
		}{
			Addr: serverUrl.Hostname(),
			Port: serverUrl.Port(),
		},
	}

	taskCh <- expectedTask

	exit := make(chan struct{})
	cancel()

	go func() {
		wg.Wait()
		close(exit)
	}()

	checkCtx, testCancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer testCancel()

	select {
	case <-checkCtx.Done():
		t.Fatal("таймаут при заврешении воркера")
	case <-exit:
		select {
		case <-resultPipe:
			t.Fatal("ошибка, воркер вернул задачу")
		default:
		}
	}
}

func TestWorkerErrResponse(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	wg := &sync.WaitGroup{}
	taskCh := make(chan *Task)
	resultPipe := make(chan *Task, 1)
	wait := make(chan struct{})
	defer func() {
		close(taskCh)
		close(resultPipe)
	}()

	wg.Add(1)
	go worker(ctx, wg, taskCh, resultPipe, 0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, _ := w.(http.Hijacker)
		net, _, _ := hj.Hijack()
		net.Close()
		close(wait)
	}))
	defer server.Close()

	serverUrl, _ := url.Parse(server.URL)

	expectedTask := &Task{
		Site:     site,
		DomainId: 1,
		Owner:    owner,
		Connection: struct {
			Addr string
			Port string
		}{
			Addr: serverUrl.Hostname(),
			Port: serverUrl.Port(),
		},
	}

	taskCh <- expectedTask
	<-wait
	task := <-resultPipe
	assert.Error(t, task.Result.Err)

	exit := make(chan struct{})

	cancel()
	go func() {
		wg.Wait()
		close(exit)
	}()

	checkCtx, testCancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer testCancel()

	select {
	case <-checkCtx.Done():
		t.Fatal("таймаут при заврешении воркера")
	case <-exit:
	}
}

func TestWorkerReturn200OK(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	wg := &sync.WaitGroup{}
	taskCh := make(chan *Task)
	resultPipe := make(chan *Task, 1)
	wait := make(chan struct{})
	defer func() {
		close(taskCh)
		close(resultPipe)
	}()

	wg.Add(1)
	go worker(ctx, wg, taskCh, resultPipe, 0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(wait)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	serverUrl, _ := url.Parse(server.URL)

	expectedTask := &Task{
		Site:     site,
		DomainId: 1,
		Owner:    owner,
		Connection: struct {
			Addr string
			Port string
		}{
			Addr: serverUrl.Hostname(),
			Port: serverUrl.Port(),
		},
	}

	taskCh <- expectedTask
	<-wait
	task := <-resultPipe
	assert.NoError(t, task.Result.Err)
	assert.Equal(t, http.StatusOK, task.Result.StatusCode)

	exit := make(chan struct{})

	cancel()
	go func() {
		wg.Wait()
		close(exit)
	}()

	checkCtx, testCancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer testCancel()

	select {
	case <-checkCtx.Done():
		t.Fatal("таймаут при заврешении воркера")
	case <-exit:
	}
}

func TestWorkerReturnNotOKStatus(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	wg := &sync.WaitGroup{}
	taskCh := make(chan *Task)
	resultPipe := make(chan *Task, 1)
	wait := make(chan struct{})
	defer func() {
		close(taskCh)
		close(resultPipe)
	}()

	wg.Add(1)
	go worker(ctx, wg, taskCh, resultPipe, 0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(wait)
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	serverUrl, _ := url.Parse(server.URL)

	expectedTask := &Task{
		Site:     site,
		DomainId: 1,
		Owner:    owner,
		Connection: struct {
			Addr string
			Port string
		}{
			Addr: serverUrl.Hostname(),
			Port: serverUrl.Port(),
		},
	}

	taskCh <- expectedTask
	<-wait
	task := <-resultPipe
	assert.NoError(t, task.Result.Err)
	assert.Equal(t, http.StatusForbidden, task.Result.StatusCode)

	exit := make(chan struct{})

	cancel()
	go func() {
		wg.Wait()
		close(exit)
	}()

	checkCtx, testCancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer testCancel()

	select {
	case <-checkCtx.Done():
		t.Fatal("таймаут при заврешении воркера")
	case <-exit:
	}
}

func TestWorkerManyTasks(t *testing.T) {
	goleak.IgnoreCurrent()
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(t.Context())
	wg := &sync.WaitGroup{}
	taskCh := make(chan *Task, 1)
	resultPipe := make(chan *Task, 1)
	before := runtime.NumGoroutine()
	defer func() {
		close(taskCh)
		close(resultPipe)
		after := runtime.NumGoroutine()
		if after > before {
			t.Errorf("горутин было %d, стало %d", before, after)
		}
	}()

	wg.Add(1)
	go worker(ctx, wg, taskCh, resultPipe, 0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	serverUrl, _ := url.Parse(server.URL)

	expectedTask := &Task{
		Site:     site,
		DomainId: 1,
		Owner:    owner,
		Connection: struct {
			Addr string
			Port string
		}{
			Addr: serverUrl.Hostname(),
			Port: serverUrl.Port(),
		},
	}

	const taskCounter = 400
	for i := range taskCounter {
		_ = i
		taskCh <- expectedTask
		<-resultPipe
	}

	exit := make(chan struct{})

	cancel()
	go func() {
		wg.Wait()
		close(exit)
	}()

	checkCtx, testCancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer testCancel()

	select {
	case <-checkCtx.Done():
		t.Fatal("таймаут при заврешении воркера")
	case <-exit:
	}

	time.Sleep(300 * time.Millisecond)
}
