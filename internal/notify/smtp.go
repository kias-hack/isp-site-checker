package notify

import (
	"context"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"
	"sync"

	"github.com/kias-hack/isp-site-checker/internal/checker"
	"github.com/kias-hack/isp-site-checker/internal/config"
)

// TODO доработать отправку, учесть ошибку и повторно пытаться отослать сообщение по домену

type domainStatus struct {
	Domain string
	Fail   bool
}

type SMTPSender struct {
	host     string
	port     int
	from     string
	to       []string
	username string
	password string
	mu       *sync.Mutex

	msg            strings.Builder
	domainStatuses map[string]*domainStatus
}

var smtpSendMail = smtp.SendMail

func NewSender(cfg *config.Config) checker.NotificationSender {
	return &SMTPSender{
		host:           cfg.SMTP.Host,
		port:           cfg.SMTP.Port,
		username:       cfg.SMTP.Username,
		password:       cfg.SMTP.Password,
		to:             []string{cfg.Recipient},
		from:           cfg.SMTP.From,
		mu:             &sync.Mutex{},
		domainStatuses: make(map[string]*domainStatus),
	}
}

func (s *SMTPSender) Send(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.msg.Len() == 0 {
		slog.Debug("нечего отправлять")
		return nil
	}

	output := make(chan error)
	defer close(output)

	go func() {
		var body strings.Builder

		body.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(s.to, ", ")))
		body.WriteString(fmt.Sprintf("Subject: %s\r\n", "Результат проверки сайтов"))
		body.WriteString("\r\n")
		body.WriteString(s.msg.String() + "\r\n")

		s.msg.Reset()

		output <- smtpSendMail(s.host, smtp.PlainAuth("", s.username, s.password, s.host), s.from, s.to, []byte(body.String()))
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("контекст завершился раньше чем письмо отправилось: %w", ctx.Err())
	case err := <-output:
		if err == nil {
			slog.Debug("уведомление отправлено")
		}
		return err
	}
}

func (s *SMTPSender) Error(domain string, result string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	status := s.getStatusFor(domain)

	if status.Fail {
		return
	}

	s.msg.WriteString(result)
	s.msg.WriteString("\r\n===================================\r\n")

	status.Fail = true
}

func (s *SMTPSender) Success(domain string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	status := s.getStatusFor(domain)

	if status.Fail {
		return
	}

	s.msg.WriteString(fmt.Sprintf("Домен %s закрыт", domain))
	s.msg.WriteString("\r\n===================================\r\n")

	status.Fail = false
}

func (s *SMTPSender) getStatusFor(domain string) *domainStatus {
	status, ok := s.domainStatuses[domain]
	if !ok {
		status = &domainStatus{
			Domain: domain,
			Fail:   false,
		}

		s.domainStatuses[domain] = status
	}

	return status
}
