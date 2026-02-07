package checker

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/kias-hack/isp-site-checker/internal/isp"
	"github.com/stretchr/testify/assert"
)

const (
	owner      = "root"
	domainName = "example.com"
	site       = "example.com"
	host       = "127.0.0.1"
	port       = "443"
)

func TestLifecycle(t *testing.T) {
	wg := &sync.WaitGroup{}

	ctx, cancel := context.WithCancel(t.Context())

	wg.Add(1)
	go scheduler(ctx, wg, make(<-chan struct{}), make(chan<- *Task), func() ([]*isp.WebDomain, error) {
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

	go scheduler(ctx, wg, ticker, taskPipe, func() ([]*isp.WebDomain, error) {
		return []*isp.WebDomain{
			{Sites: []string{"example.com"}},
		}, fmt.Errorf("test error")
	})

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

func TestSendTasks(t *testing.T) {
	wg := &sync.WaitGroup{}
	ticker := make(chan struct{})
	taskPipe := make(chan *Task)
	defer func() {
		close(ticker)
		close(taskPipe)
	}()

	testCases := []struct {
		name    string
		domains []*isp.WebDomain
		tasks   []*Task
	}{
		{
			name: "Один домен и один сайт",
			domains: []*isp.WebDomain{
				{Id: 1, Name: domainName, Owner: owner, IPAddr: host, Port: port, Sites: []string{domainName}},
			},
			tasks: []*Task{
				{
					DomainId:   1,
					Owner:      owner,
					DomainName: domainName,
					Site:       domainName,
					Connection: struct {
						Addr string
						Port string
					}{
						Addr: host,
						Port: port,
					},
				},
			},
		},
		{
			name: "Один домен и два сайта",
			domains: []*isp.WebDomain{
				{Id: 1, Name: domainName, Owner: owner, IPAddr: host, Port: port, Sites: []string{domainName, "www." + domainName}},
			},
			tasks: []*Task{
				{
					DomainId:   1,
					Owner:      owner,
					DomainName: domainName,
					Site:       domainName,
					Connection: struct {
						Addr string
						Port string
					}{
						Addr: host,
						Port: port,
					},
				},
				{
					DomainId:   1,
					Owner:      owner,
					DomainName: domainName,
					Site:       "www." + domainName,
					Connection: struct {
						Addr string
						Port string
					}{
						Addr: host,
						Port: port,
					},
				},
			},
		},
		{
			name: "два домена и 4 сайта в итоге",
			domains: []*isp.WebDomain{
				{Id: 1, Name: domainName, Owner: owner, IPAddr: host, Port: port, Sites: []string{domainName, "www." + domainName}},
				{Id: 2, Name: "test.test", Owner: "test", IPAddr: host, Port: port, Sites: []string{"test.test", "www.test.test"}},
			},
			tasks: []*Task{
				{
					DomainId:   1,
					Owner:      owner,
					DomainName: domainName,
					Site:       domainName,
					Connection: struct {
						Addr string
						Port string
					}{
						Addr: host,
						Port: port,
					},
				},
				{
					DomainId:   1,
					Owner:      owner,
					DomainName: domainName,
					Site:       "www." + domainName,
					Connection: struct {
						Addr string
						Port string
					}{
						Addr: host,
						Port: port,
					},
				},
				{
					DomainId:   2,
					Owner:      "test",
					DomainName: "test.test",
					Site:       "test.test",
					Connection: struct {
						Addr string
						Port string
					}{
						Addr: host,
						Port: port,
					},
				},
				{
					DomainId:   2,
					Owner:      "test",
					DomainName: "test.test",
					Site:       "www.test.test",
					Connection: struct {
						Addr string
						Port string
					}{
						Addr: host,
						Port: port,
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			wg.Add(1)
			go scheduler(ctx, wg, ticker, taskPipe, func() ([]*isp.WebDomain, error) {
				return testCase.domains, nil
			})

			ticker <- struct{}{}
			timer := time.NewTimer(1 * time.Second)

			for _, expectedTask := range testCase.tasks {
				select {
				case <-timer.C:
					t.Fatal("таймаут при прогоне теста")
				case actualTask := <-taskPipe:
					assert.Equal(t, actualTask, expectedTask)
				}
			}

			select {
			case <-taskPipe:
				t.Fatal("ошибка, добавлена лишняя задача")
			default:
			}

		})
	}

	wait := make(chan struct{})
	timer := time.NewTimer(200 * time.Millisecond)
	defer timer.Stop()
	go func() {
		wg.Wait()
		close(wait)
	}()
	select {
	case <-wait:
	case <-timer.C:
		t.Fatal("timeout")
	}
}
