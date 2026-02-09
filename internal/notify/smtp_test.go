package notify

import (
	"context"
	"fmt"
	"net/smtp"
	"testing"
	"time"

	"github.com/kias-hack/isp-site-checker/internal/util"
	"github.com/stretchr/testify/assert"
)

func TestMailTesterSendMail(t *testing.T) {
	testMail := &util.Mail{
		From:    "test@test.ru",
		To:      []string{"test@test.ru"},
		Subject: "Test",
		Message: "Message body",
	}

	mailBody, _ := util.BuildMailBody(testMail)
	expectedAuth := smtp.PlainAuth("", "username", "password", "localhost")
	expectedAddr := "localhost:23"

	smtpSendMail := func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		assert.Equal(t, []byte(mailBody), msg)
		assert.Equal(t, to, testMail.To)
		assert.Equal(t, from, testMail.From)
		assert.Equal(t, expectedAuth, a)
		assert.Equal(t, expectedAddr, addr)

		return nil
	}

	sender := &mailSender{
		addr:     expectedAddr,
		auth:     expectedAuth,
		smtpFunc: smtpSendMail,
	}

	assert.NoError(t, sender.Send(t.Context(), testMail))
}

func TestMailTesterSendMailWithError(t *testing.T) {
	testMail := &util.Mail{
		From:    "test@test.ru",
		To:      []string{"test@test.ru"},
		Subject: "Test",
		Message: "Message body",
	}

	expectedAuth := smtp.PlainAuth("", "username", "password", "localhost")
	expectedAddr := "localhost:23"

	smtpSendMail := func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		return fmt.Errorf("some error")
	}

	sender := &mailSender{
		addr:     expectedAddr,
		auth:     expectedAuth,
		smtpFunc: smtpSendMail,
	}

	assert.Error(t, sender.Send(t.Context(), testMail))
}

func TestMailTesterSendMailWithStopByContext(t *testing.T) {
	testMail := &util.Mail{
		From:    "test@test.ru",
		To:      []string{"test@test.ru"},
		Subject: "Test",
		Message: "Message body",
	}

	expectedAuth := smtp.PlainAuth("", "username", "password", "localhost")
	expectedAddr := "localhost:23"

	smtpSendMail := func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		time.Sleep(1 * time.Second)

		return nil
	}

	sender := &mailSender{
		addr:     expectedAddr,
		auth:     expectedAuth,
		smtpFunc: smtpSendMail,
	}

	ctx, cancel := context.WithCancel(t.Context())

	exit := make(chan struct{})

	go func() {
		sender.Send(ctx, testMail)
		close(exit)
	}()

	cancel()

	timeoutCtx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
	defer cancel()

	select {
	case <-exit:
	case <-timeoutCtx.Done():
		t.Fatal("timeout")
	}
}
