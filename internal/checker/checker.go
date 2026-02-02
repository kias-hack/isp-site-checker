package checker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/kias-hack/isp-site-checker/internal/config"
	"github.com/kias-hack/isp-site-checker/internal/isp"
	"github.com/kias-hack/isp-site-checker/internal/notify"
)

var getDomains = isp.GetWebDomain

type Task struct {
	domainInfo *isp.WebDomain
	domain     string
	Result     struct {
		StatusCode int
		Err        error
		Timestamp  time.Time
	}
}

type Checker struct {
	config *config.Config
	ctx    context.Context
	work   bool
	cancel context.CancelFunc
	wg     *sync.WaitGroup

	taskPipe   chan *Task
	resultPipe chan *Task

	notifier notify.Notifier
}

func NewChecker(config *config.Config, notifier notify.Notifier) *Checker {
	return &Checker{
		config:     config,
		wg:         &sync.WaitGroup{},
		taskPipe:   make(chan *Task),
		resultPipe: make(chan *Task),
		work:       false,
		notifier:   notifier,
	}
}

func (c *Checker) resultHandler() {
	defer c.wg.Done()

	for {
		select {
		case task := <-c.resultPipe:
			logger := slog.With("component", "resultHandler")

			if task.Result.Err == nil && task.Result.StatusCode == http.StatusForbidden {
				c.notifier.Success(task.domain, fmt.Sprintf("Домен %s закрыт - %d\r\n", task.domain, task.Result.StatusCode))
				logger.Debug("получен результат, сайт закрыт", "domain", task.domain)
				continue
			}

			logger.Debug("получен результат, начинаю обрабатывать результат", "domain", task.domain)

			msg := strings.Builder{}

			msg.WriteString("Проверка домена выявила проблему\n")
			msg.WriteString(fmt.Sprintf("Домен: %s\n", task.domain))
			msg.WriteString(fmt.Sprintf("Владелец: %s\n", task.domainInfo.Owner))
			msg.WriteString(fmt.Sprintf("Время: %s\n", task.Result.Timestamp))
			msg.WriteString("\n")

			if task.Result.Err != nil {
				logger.Debug("ошибка в результате", "domain", task.domain)
				msg.WriteString(fmt.Sprintf("Произошла ошибка - %s", task.Result.Err.Error()))
			} else {
				logger.Debug("неверный статус в результате", "domain", task.Result.StatusCode)
				msg.WriteString(fmt.Sprintf("Код ответа - %d", task.Result.StatusCode))
			}

			c.notifier.Fail(task.domain, msg.String())

			logger.Info("обработан результат по домену", "domain", task.domain)
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *Checker) sheduler() {
	ticker := time.NewTicker(c.config.ScrapeInterval)

	defer ticker.Stop()
	defer c.wg.Done()

	for {
		select {
		case <-ticker.C:
			slog.Debug("начинаю проверку доменов, получаю список доменов")

			domains, err := getDomains(c.config.MgrCtlPath)
			if err != nil {
				slog.Error("при получении списка доменов из ISPManager произошла ошибка", "err", err)
				continue
			}

			for _, domainInfo := range domains {
				logger := slog.With("component", "scheduler", "name", domainInfo.Name, "owner", domainInfo.Owner)

				if !domainInfo.Active {
					logger.Debug("домен отключен, пропускаю")
					continue
				}

				logger.Debug("начало проверки домена, получаю список поддоменов")

				domainsForCheck := findSubdomain(domainInfo.Owner, domainInfo.Name)

				for _, domain := range domainsForCheck {
					logger.Debug("отправлена задача на обработку", "domain", domain)

					c.taskPipe <- &Task{
						domainInfo: domainInfo,
						domain:     domain,
					}
				}
			}
		case <-c.ctx.Done():
			c.work = false
			return
		}
	}
}

func (c *Checker) worker(n int) {
	defer c.wg.Done()

	slog.Debug("worker started", "id", n)

	for {
		select {
		case <-c.ctx.Done():
			return
		case task := <-c.taskPipe:
			logger := slog.With("component", fmt.Sprintf("worker[%d]", n))

			logger.Debug("получена задача для обработки", "task", task)

			client := createClient(task.domainInfo.IPAddr, task.domainInfo.Port)
			defer client.CloseIdleConnections()

			url := fmt.Sprintf("http://%s/", task.domain)

			task.Result.Timestamp = time.Now()

			// TODO внедрить контекст
			resp, err := client.Get(url)

			if err != nil {
				logger.Debug("произошла ошибка при подключении к серверу", "domain", task.domain, "err", err)
				task.Result.Err = err
			} else {
				logger.Debug("получен статус от сервера", "domain", task.domain, "status", resp.StatusCode)
				task.Result.StatusCode = resp.StatusCode
			}

			c.resultPipe <- task
		}
	}
}

func (c *Checker) Start() error {
	if c.work {
		return errors.New("процесс уже запущен")
	}

	ctx, cancel := context.WithCancel(context.Background())
	c.ctx = ctx
	c.cancel = cancel
	c.work = true

	c.wg.Add(2)
	go c.resultHandler()
	go c.sheduler()

	for n := range 10 {
		c.wg.Add(1)
		go c.worker(n)
	}

	return nil
}

func (c *Checker) Stop(ctx context.Context) error {
	defer func() {
		close(c.taskPipe)
		close(c.resultPipe)
	}()

	c.cancel()

	waitChan := make(chan interface{})

	go func() {
		c.wg.Wait()
		close(waitChan)
	}()

	select {
	case <-waitChan:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("контекст завершился пока я ждал завершения чекера %w", ctx.Err())
	}
}
