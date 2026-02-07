package checker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/kias-hack/isp-site-checker/internal/config"
	"github.com/kias-hack/isp-site-checker/internal/isp"
	"github.com/kias-hack/isp-site-checker/internal/notify"
)

const workerPoolCountDefault int = 10

type Task struct {
	DomainId   int
	Owner      string
	DomainName string
	Site       string
	Connection struct {
		Addr string
		Port string
	}
	Result struct {
		StatusCode int
		Err        error
		Timestamp  time.Time
	}
}

type Checker struct {
	config *config.Config
	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup
	work   bool

	taskPipe    chan *Task
	resultPipe  chan *Task
	schedTicker chan struct{}

	getDomains isp.GetWebDomainsFunc

	notifier notify.Notifier
}

func NewChecker(config *config.Config, notifier notify.Notifier, getDomains isp.GetWebDomainsFunc) *Checker {
	return &Checker{
		config:     config,
		wg:         &sync.WaitGroup{},
		notifier:   notifier,
		getDomains: getDomains,
		work:       false,
	}
}

func (c *Checker) Start() error {
	if c.ctx != nil {
		return errors.New("process already running")
	}

	ctx, cancel := context.WithCancel(context.Background())
	c.ctx = ctx
	c.cancel = cancel
	c.taskPipe = make(chan *Task)
	c.resultPipe = make(chan *Task)
	c.schedTicker = make(chan struct{})

	c.wg.Add(3)
	go resultHandler(ctx, c.wg, c.resultPipe, c.notifier)

	go func() {
		defer c.wg.Done()
		ticker := time.NewTicker(c.config.ScrapeInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.schedTicker <- struct{}{}
			case <-c.ctx.Done():
				return
			}
		}
	}()

	go scheduler(ctx, c.wg, c.schedTicker, c.taskPipe, c.getDomains)

	for n := range workerPoolCountDefault {
		c.wg.Add(1)
		go worker(c.ctx, c.wg, c.taskPipe, c.resultPipe, n)
	}

	c.work = true

	return nil
}

func (c *Checker) Stop(ctx context.Context) error {
	if !c.work {
		return nil
	}
	defer func() {
		close(c.taskPipe)
		close(c.resultPipe)
	}()

	var notifierErr error
	if err := c.notifier.Stop(ctx); err != nil {
		notifierErr = fmt.Errorf("notifier stop: %w", err)
	}

	c.cancel()

	waitChan := make(chan struct{})

	go func() {
		c.wg.Wait()
		close(waitChan)
	}()

	var waitErr error
	select {
	case <-waitChan:
		c.work = false
	case <-ctx.Done():
		waitErr = fmt.Errorf("checker wait: %w", ctx.Err())
	}

	return errors.Join(notifierErr, waitErr)
}
