package notify

import (
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"net/mail"
	"net/smtp"
	"strings"

	"golang.org/x/net/idna"
)

type SmtpSendMailFunc func(addr string, a smtp.Auth, from string, to []string, msg []byte) error

type Mail struct {
	From    string
	To      []string
	Subject string
	Message string
}

type MailSender interface {
	Send(ctx context.Context, mail *Mail) error
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

func (m *mailSender) Send(ctx context.Context, mail *Mail) error {
	body, err := buildMailBody(mail)
	if err != nil {
		return fmt.Errorf("ошибка при формировании тела письма: %w", err)
	}

	errChan := make(chan error)

	go func() {
		errChan <- m.smtpFunc(m.addr, m.auth, mail.From, mail.To, []byte(body))
		close(errChan)
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func buildMailBody(email *Mail) (string, error) {
	builder := strings.Builder{}

	if email.From == "" {
		return "", fmt.Errorf("sender can`t be empty")
	}

	from, err := mail.ParseAddress(email.From)
	if err != nil {
		return "", fmt.Errorf("ошибка парсинга отправителя: %w", err)
	}

	var fromStr string

	if from.Name == "" {
		fromStr = from.Address
	} else {
		chunks := strings.Split(from.Address, "@")

		punyDomain, err := idna.Lookup.ToASCII(chunks[1])
		if err != nil {
			return "", fmt.Errorf("ошибка кодирования домена: %s", err)
		}

		fromStr = fmt.Sprintf("%s <%s>", mime.BEncoding.Encode("UTF-8", from.Name), fmt.Sprintf("%s@%s", chunks[0], punyDomain))
	}

	builder.WriteString(fmt.Sprintf("From: %s\n", fromStr))

	if len(email.To) == 0 {
		return "", fmt.Errorf("receiver can`t be empty")
	}

	var toList []string
	for _, address := range email.To {
		toAddress, err := mail.ParseAddress(address)
		if err != nil {
			return "", fmt.Errorf("ошибка парсинга адреса получателя: %w", err)
		}

		chunks := strings.Split(toAddress.Address, "@")

		receiverPunycode, err := idna.Lookup.ToASCII(chunks[1])
		if err != nil {
			return "", fmt.Errorf("ошибка кодирования получателя: %w", err)
		}

		toList = append(toList, fmt.Sprintf("%s@%s", chunks[0], receiverPunycode))
	}

	builder.WriteString(fmt.Sprintf("To: %s\n", strings.Join(toList, ", ")))

	if email.Subject == "" {
		return "", fmt.Errorf("subject can`t be empty")
	}

	builder.WriteString(fmt.Sprintf("Subject: %s\n", mime.BEncoding.Encode("UTF-8", email.Subject)))
	builder.WriteString("Content-Type: text/plain; charset=UTF-8\n")
	builder.WriteString("Content-Type-Encoding: base64\n")

	if email.Message == "" {
		return "", fmt.Errorf("body can`t be empty")
	}

	builder.WriteString(fmt.Sprintf("\n%s", base64.StdEncoding.EncodeToString([]byte(email.Message))))

	return builder.String(), nil
}
