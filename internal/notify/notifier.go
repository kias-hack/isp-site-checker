package notify

//go:generate mockgen -source=notifier.go -destination=mock_notifier.go -package=notify

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/kias-hack/isp-site-checker/internal/config"
	"github.com/kias-hack/isp-site-checker/internal/util"
)

type siteStatus string

const (
	Fail    siteStatus = "fail"
	Success siteStatus = "success"
)

const SiteRetentionPeriod = 24 * time.Hour

type SiteNotification struct {
	Site        string
	Status      siteStatus
	Message     string
	NeedNotify  bool
	LastSended  time.Time
	LastUpdated time.Time
}

type Notifier interface {
	Success(site string, message string)
	Fail(site string, message string)
	Stop(context.Context) error
}

func NewNotifier(cfg *config.Config, mailSender MailSender) Notifier {
	n := &notifier{
		wg:             &sync.WaitGroup{},
		timeout:        cfg.SendTimeout,
		interval:       cfg.SendInterval,
		ticker:         make(chan struct{}),
		mailSender:     mailSender,
		repeatInterval: cfg.RepeatInterval,
		mailSettings: struct {
			From    string
			To      []string
			Subject string
		}{
			From:    cfg.EMail.From,
			To:      cfg.EMail.To,
			Subject: cfg.EMail.Subject,
		},
		sitesMap: make(map[string]*SiteNotification),
	}

	n.start()

	return n
}

type notifier struct {
	wg             *sync.WaitGroup
	timeout        time.Duration
	interval       time.Duration
	repeatInterval time.Duration
	stop           chan struct{}
	mailSender     MailSender
	ticker         chan struct{}

	mu sync.Mutex

	mailSettings struct {
		From    string
		To      []string
		Subject string
	}

	sitesMap map[string]*SiteNotification
}

func (n *notifier) Success(site string, message string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	siteInfo := n.getSite(site)

	siteInfo.LastUpdated = time.Now()

	if siteInfo.Status != Success {
		slog.Info("site closed", "site", site)

		siteInfo.NeedNotify = true
		siteInfo.Message = message
		siteInfo.Status = Success
	}
}

func (n *notifier) Fail(site string, message string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	siteInfo := n.getSite(site)

	siteInfo.LastUpdated = time.Now()

	if siteInfo.Status != Fail {
		slog.Info("site returned invalid response", "site", site)

		siteInfo.NeedNotify = true
		siteInfo.Message = message
		siteInfo.Status = Fail
	}
}

func (n *notifier) getSite(site string) *SiteNotification {
	info, ok := n.sitesMap[site]
	if ok {
		return info
	}

	newInfo := &SiteNotification{
		Site:       site,
		Status:     Success,
		NeedNotify: false,
		LastSended: time.Now(),
	}

	n.sitesMap[site] = newInfo

	return newInfo
}

func (n *notifier) Stop(ctx context.Context) error {
	if n.stop == nil {
		return nil
	}

	close(n.stop)

	wait := make(chan struct{})
	go func() {
		n.wg.Wait()
		close(wait)
	}()

	select {
	case <-wait:
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

func (n *notifier) start() {
	if n.stop != nil {
		panic("service already started")
	}

	n.stop = make(chan struct{})
	n.wg.Add(2)

	go func() {
		defer n.wg.Done()
		ticker := time.NewTicker(n.interval)

		defer ticker.Stop()
		defer close(n.ticker)

		for {
			select {
			case <-n.stop:
				return
			case <-ticker.C:
				n.ticker <- struct{}{}
			}
		}
	}()

	go n.worker()
}

func (n *notifier) worker() {
	defer n.wg.Done()

	for {
		select {
		case <-n.ticker:
			var toNotify []struct {
				site    string
				message string
			}

			n.mu.Lock()
			for site, info := range n.sitesMap {
				if time.Since(info.LastUpdated) >= SiteRetentionPeriod && info.Status != Fail {
					slog.Debug("cleaning up site record", "site", site, "period", time.Since(info.LastSended))
					delete(n.sitesMap, site)
				}

				if n.canSendMail(info) {
					toNotify = append(toNotify, struct {
						site    string
						message string
					}{
						site:    site,
						message: info.Message,
					})
				}
			}
			n.mu.Unlock()

			if len(toNotify) == 0 {
				continue
			}

			message := strings.Builder{}
			for _, item := range toNotify {
				message.WriteString(item.message)
				message.WriteString("\r\n=============================\r\n")
			}

			ctx, cancel := context.WithTimeout(context.Background(), n.timeout)

			if err := n.mailSender.Send(ctx, &util.Mail{
				Subject: n.mailSettings.Subject,
				From:    n.mailSettings.From,
				To:      n.mailSettings.To,
				Message: message.String(),
			}); err != nil {
				slog.Error("notification send failed", "err", err)
			} else {
				for _, item := range toNotify {
					n.mu.Lock()
					info, ok := n.sitesMap[item.site]
					if ok {
						info.NeedNotify = false
						info.LastSended = time.Now()
						info.LastUpdated = time.Now()
					} else {
						slog.Warn("sent notify by deleted site row, strange")
					}
					n.mu.Unlock()
				}
			}

			cancel()
		case <-n.stop:
			return
		}
	}
}

func (n *notifier) canSendMail(info *SiteNotification) bool {
	if info.NeedNotify {
		return true
	}

	slog.Debug("checking repeat notification send", "info", info, "since", time.Since(info.LastSended))

	if info.Status == Fail && time.Since(info.LastSended) >= n.repeatInterval {
		return true
	}

	return false
}
