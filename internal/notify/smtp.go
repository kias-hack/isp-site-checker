package notify

import (
	"context"
	"fmt"
	"log/slog"
	"net/mail"
	"net/smtp"

	"github.com/kias-hack/isp-site-checker/internal/util"
)

//go:generate mockgen -source=smtp.go -destination=mock_smtp.go -package=notify

type SmtpSendMailFunc func(addr string, a smtp.Auth, from string, to []string, msg []byte) error

type MailSender interface {
	Send(ctx context.Context, mail *util.Mail) error
}

func NewMailSender(host string, port string, username string, password string, smtpFunc SmtpSendMailFunc) MailSender {
	return &mailSender{
		addr:     fmt.Sprintf("%s:%s", host, port),
		auth:     smtp.PlainAuth("", username, password, host),
		smtpFunc: smtpFunc,
	}
}

type mailSender struct {
	addr     string
	auth     smtp.Auth
	smtpFunc SmtpSendMailFunc
}

func (m *mailSender) Send(ctx context.Context, email *util.Mail) error {
	body, err := util.BuildMailBody(email)
	if err != nil {
		return fmt.Errorf("failed to build mail body: %w", err)
	}

	errChan := make(chan error)

	go func() {
		defer close(errChan)
		slog.Debug("sending mail", "msg", body)
		address, err := mail.ParseAddress(email.From)
		if err != nil {
			errChan <- err
			return
		}
		errChan <- m.smtpFunc(m.addr, m.auth, address.Address, email.To, []byte(body))
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
