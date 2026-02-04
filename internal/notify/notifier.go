package notify

//go:generate mockgen -source=notifier.go -destination=mock_notifier.go -package=notify

import (
	"log/slog"
	"sync"
	"time"

	"github.com/kias-hack/isp-site-checker/internal/config"
)

type Result struct {
	Domain     string
	Owner      string
	StatusCode int
	Err        error
	Timestamp  time.Time
}

type Notifier interface {
	Success(site string, message string)
	Fail(site string, message string)
}

func NewNotifier(cfg *config.Config) Notifier {
	n := &notifier{
		wg:       &sync.WaitGroup{},
		timeout:  cfg.SMTP.SendTimeout,
		interval: cfg.SMTP.SendInterval,
	}

	n.start()

	return n
}

type notifier struct {
	wg       *sync.WaitGroup
	timeout  time.Duration
	interval time.Duration
	stop     chan struct{}
}

func (n *notifier) Success(domain string, message string) {

}

func (n *notifier) Fail(domain string, message string) {

}

func (n *notifier) start() {
	if n.stop == nil {
		panic("сервис уже запущен")
	}

	n.stop = make(chan struct{})

	go func() {
		ticker := time.NewTicker(n.interval)

		defer ticker.Stop()
		defer n.wg.Done()

		for {
			select {
			case <-ticker.C:
				// ctx, cancel := context.WithTimeout(c.ctx, 2*time.Second)
				// defer cancel()

				slog.Debug("отправляю уведомление")
				// if err := n.send(ctx); err != nil {
				// 	slog.Error("ошибка отправки увдомлений", "err", err)
				// }
			case <-n.stop:
				return
			}
		}
	}()
}

func (n *notifier) Close() {
	defer close(n.stop)

}
