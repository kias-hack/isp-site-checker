package util

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log/slog"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strings"

	"golang.org/x/net/idna"
)

type MXResolverFunc func(email string) (host string, err error)

type Mail struct {
	From    string
	To      []string
	Subject string
	Message string
}

func GetUsernameAndHostByEmailAddress(email string) (smtpHost string, err error) {
	mailAddress, err := mail.ParseAddress(email)
	if err != nil {
		return "", fmt.Errorf("failed parse email address: %w", err)
	}

	chunks := strings.Split(mailAddress.Address, "@")
	host := chunks[1]

	slog.Info("trying lookup smtp subdomain", "host", host)
	if _, err = net.LookupIP("smtp." + host); err == nil {
		smtpHost = "smtp." + host

		return
	}

	slog.Info("smtp subdomain not found, trying fetch mx record", "host", host)

	mxList, err := net.LookupMX(host)
	if err != nil {
		return "", fmt.Errorf("failed lookup mx: %w", err)
	}

	smtpHost = strings.TrimSuffix(mxList[0].Host, ".")

	slog.Info("found mx", "mx", smtpHost)

	return
}

func BuildMailBody(email *Mail) (string, error) {
	builder := strings.Builder{}

	if email.From == "" {
		return "", fmt.Errorf("sender can`t be empty")
	}

	from, err := mail.ParseAddress(email.From)
	if err != nil {
		return "", fmt.Errorf("failed to parse sender address: %w", err)
	}

	var fromStr string

	if from.Name == "" {
		fromStr = from.Address
	} else {
		chunks := strings.Split(from.Address, "@")

		punyDomain, err := idna.Lookup.ToASCII(chunks[1])
		if err != nil {
			return "", fmt.Errorf("failed to encode domain: %s", err)
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
			return "", fmt.Errorf("failed to parse recipient address: %w", err)
		}

		chunks := strings.Split(toAddress.Address, "@")

		receiverPunycode, err := idna.Lookup.ToASCII(chunks[1])
		if err != nil {
			return "", fmt.Errorf("failed to encode recipient: %w", err)
		}

		toList = append(toList, fmt.Sprintf("%s@%s", chunks[0], receiverPunycode))
	}

	builder.WriteString(fmt.Sprintf("To: %s\n", strings.Join(toList, ", ")))

	if email.Subject == "" {
		return "", fmt.Errorf("subject can`t be empty")
	}

	builder.WriteString(fmt.Sprintf("Subject: %s\n", mime.BEncoding.Encode("UTF-8", email.Subject)))
	builder.WriteString("Content-Type: text/plain; charset=UTF-8\n")
	builder.WriteString("Content-Transfer-Encoding: base64\n")

	if email.Message == "" {
		return "", fmt.Errorf("body can`t be empty")
	}

	builder.WriteString("\n")
	builder.WriteString(base64.StdEncoding.EncodeToString([]byte(email.Message)))

	return builder.String(), nil
}

func SendMail(addr string, a smtp.Auth, from string, to []string, msg []byte, skipVerify bool) error {
	c, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	defer c.Close()

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	client, err := smtp.NewClient(c, host)
	if err != nil {
		return err
	}
	defer client.Close()

	// if port != "25" {
	tlsConfig := &tls.Config{ServerName: host}
	if skipVerify {
		tlsConfig.InsecureSkipVerify = true
	}
	if err := client.StartTLS(tlsConfig); err != nil {
		slog.Debug("starttls failed, sending without TLS", "err", err)
	}
	// }

	if a != nil {
		if err := client.Auth(a); err != nil {
			return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, rcpt := range to {
		if err = client.Rcpt(rcpt); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}

	return client.Quit()
}
