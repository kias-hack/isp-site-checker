package checker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/smtp"
	"strings"
	"sync"
	"time"

	"github.com/kias-hack/isp-site-checker/internal/config"
	"github.com/kias-hack/isp-site-checker/internal/isp"
)

var sendMail = smtp.SendMail
var getDomains = isp.GetWebDomain

type CheckInfo struct {
	Domain     string
	StatusCode int
	Owner      string
	Timestamp  time.Time
	Err        error
}

type Checker struct {
	config *config.Config
	ctx    context.Context
	work   bool
	cancel context.CancelFunc
	wg     *sync.WaitGroup

	resultPipe chan CheckInfo
}

func NewChecker(config *config.Config) *Checker {
	ctx, cncl := context.WithCancel(context.Background())

	sendMail = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		return nil
	}

	return &Checker{
		ctx:        ctx,
		cancel:     cncl,
		config:     config,
		wg:         &sync.WaitGroup{},
		resultPipe: make(chan CheckInfo, 10),
		work:       false,
	}
}

func (c *Checker) notifyWorker() {
	defer c.wg.Done()

	for {
		select {
		case result := <-c.resultPipe:
			logger := slog.With("domain", result.Domain, "err", result.Err, "status", result.StatusCode)

			if result.Err == nil && result.StatusCode == http.StatusForbidden {
				logger.Debug("получен результат, пропускаю")
				continue
			}

			logger.Debug("получен результат, отправлять начинаю уведомление")

			msg := strings.Builder{}

			msg.WriteString("Проверка домена выявила проблему\n")
			msg.WriteString(fmt.Sprintf("Домен: %s\n", result.Domain))
			msg.WriteString(fmt.Sprintf("Владелец: %s\n", result.Owner))
			msg.WriteString(fmt.Sprintf("Время: %s\n", result.Timestamp))
			msg.WriteString("\n")

			if result.Err != nil {
				logger.Debug("ошибка в результате")
				msg.WriteString(fmt.Sprintf("Произошла ошибка - %s", result.Err.Error()))
			} else {
				logger.Debug("неверный статус в результате")
				msg.WriteString(fmt.Sprintf("Код ответа - %d", result.StatusCode))
			}

			auth := smtp.PlainAuth("", c.config.SMTP.Username, c.config.SMTP.Password, c.config.SMTP.Host)
			addr := fmt.Sprintf("%s:%d", c.config.SMTP.Host, c.config.SMTP.Port)

			logger.Debug("отправляю письмо")

			// TODO доработать отправку почты
			err := sendMail(addr, auth, c.config.SMTP.From, []string{c.config.Recipient}, []byte(msg.String()))

			if err != nil {
				logger.Error("ошибка отправки по SMTP", "err", err)
				continue
			}

			logger.Debug("письмо отправлено")

			logger.Info("обработан результат по домену")
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *Checker) Start() error {
	if c.work {
		return errors.New("процесс уже запущен")
	}

	c.work = true

	c.wg.Add(2)

	go c.notifyWorker()

	go func() {
		ticker := time.NewTicker(c.config.ScrapeInterval)

		defer ticker.Stop()
		defer c.wg.Done()

		for {
			select {
			case <-ticker.C:
				c.checkDomains()
			case <-c.ctx.Done():
				c.work = false
				return
			}
		}
	}()

	return nil
}

func (c *Checker) Stop(ctx context.Context) error {
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

func (c *Checker) checkDomains() {
	slog.Debug("начинаю проверку доменов, получаю список доменов")

	domains, err := getDomains(c.config.MgrCtlPath)
	if err != nil {
		slog.Error("при получении списка доменов из ISPManager произошла ошибка", "err", err)
		return
	}

	for _, domainInfo := range domains {
		if !domainInfo.Active {
			slog.Debug("домен отключен", "domain", domainInfo.Name, "owner", domainInfo.Owner)
			continue
		}

		c.wg.Add(1)
		go func(domainInfo *isp.WebDomain) {
			defer c.wg.Done()

			logger := slog.With("id", domainInfo.Id, "name", domainInfo.Name, "addr", domainInfo.IPAddr)

			logger.Info("начало проверки домена, получаю список поддоменов")

			domainsForCheck := findSubdomain(domainInfo.Owner, domainInfo.Name)

			client := createClient(domainInfo.IPAddr, domainInfo.Port)

			for _, domain := range domainsForCheck {
				logger.Debug("проверка домена", "domain", domain)

				url := fmt.Sprintf("http://%s/", domain)

				result := CheckInfo{
					Domain:    domain,
					Owner:     domainInfo.Owner,
					Timestamp: time.Now(),
				}

				// TODO внедрить контекст
				resp, err := client.Get(url)

				if err != nil {
					logger.Debug("произошла ошибка при подключении к серверу", "domain", domain)
					result.Err = err
				} else {
					logger.Debug("получен статус от сервера", "domain", domain)
					result.StatusCode = resp.StatusCode
				}

				c.resultPipe <- result

				logger.Debug("отправлен результат", "domain", domain)
			}
		}(domainInfo)
	}
}
