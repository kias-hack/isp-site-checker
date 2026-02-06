package notify

import (
	"context"
	"fmt"
	"net/smtp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBuildMailBody(t *testing.T) {
	testCases := []struct {
		name        string
		mail        *Mail
		expectedOut string
		errText     string
		isErr       bool
	}{
		{
			name: "Пустой отправитель",
			mail: &Mail{
				To:      []string{"s@example.com"},
				Subject: "test",
				Message: "test",
			},
			expectedOut: "",
			errText:     "sender can`t be empty",
			isErr:       true,
		},
		{
			name: "Ноль получателей",
			mail: &Mail{
				From:    "me@example.com",
				To:      []string{},
				Subject: "test",
				Message: "test",
			},
			expectedOut: "",
			errText:     "receiver can`t be empty",
			isErr:       true,
		},
		{
			name: "Пустой получатель",
			mail: &Mail{
				From:    "me@example.com",
				To:      nil,
				Subject: "test",
				Message: "test",
			},
			expectedOut: "",
			errText:     "receiver can`t be empty",
			isErr:       true,
		},
		{
			name: "Пустая тема",
			mail: &Mail{
				From:    "me@example.com",
				To:      []string{"set@example.com"},
				Subject: "",
				Message: "test",
			},
			expectedOut: "",
			errText:     "subject can`t be empty",
			isErr:       true,
		},
		{
			name: "Пустое тело",
			mail: &Mail{
				From:    "me@example.com",
				To:      []string{"s@example.com"},
				Subject: "test",
				Message: "",
			},
			expectedOut: "",
			errText:     "body can`t be empty",
			isErr:       true,
		},
		{
			name: "Успешное создание",
			mail: &Mail{
				From:    "me@example.com",
				To:      []string{"s@example.com"},
				Subject: "test",
				Message: "Test",
			},
			expectedOut: `From: me@example.com
To: s@example.com
Subject: test
Content-Type: text/plain; charset=UTF-8
Content-Type-Encoding: base64

VGVzdA==`,
			errText: "",
			isErr:   false,
		},
		{
			name: "Успешное создание и несколько получателей",
			mail: &Mail{
				From:    "me@example.com",
				To:      []string{"s@example.com", "w@example.com"},
				Subject: "test",
				Message: "Test",
			},
			expectedOut: `From: me@example.com
To: s@example.com, w@example.com
Subject: test
Content-Type: text/plain; charset=UTF-8
Content-Type-Encoding: base64

VGVzdA==`,
			errText: "",
			isErr:   false,
		},
		{
			name: "Тема с русскими символами",
			mail: &Mail{
				From:    "me@example.com",
				To:      []string{"s@example.com"},
				Subject: "test с русскими символами",
				Message: "Test",
			},
			expectedOut: `From: me@example.com
To: s@example.com
Subject: =?UTF-8?b?dGVzdCDRgSDRgNGD0YHRgdC60LjQvNC4INGB0LjQvNCy0L7Qu9Cw0LzQuA==?=
Content-Type: text/plain; charset=UTF-8
Content-Type-Encoding: base64

VGVzdA==`,
			errText: "",
			isErr:   false,
		},
		{
			name: "Письмо с отправителем с доп. названием",
			mail: &Mail{
				From:    "Отправитель письма <sender@example.com>",
				To:      []string{"s@example.com"},
				Subject: "test с русскими символами",
				Message: "Test",
			},
			expectedOut: `From: =?UTF-8?b?0J7RgtC/0YDQsNCy0LjRgtC10LvRjCDQv9C40YHRjNC80LA=?= <sender@example.com>
To: s@example.com
Subject: =?UTF-8?b?dGVzdCDRgSDRgNGD0YHRgdC60LjQvNC4INGB0LjQvNCy0L7Qu9Cw0LzQuA==?=
Content-Type: text/plain; charset=UTF-8
Content-Type-Encoding: base64

VGVzdA==`,
			errText: "",
			isErr:   false,
		},
		{
			name: "В отправителе русский домен",
			mail: &Mail{
				From:    "Отправитель письма <sender@тест.рф>",
				To:      []string{"s@example.com"},
				Subject: "test с русскими символами",
				Message: "Test",
			},
			expectedOut: `From: =?UTF-8?b?0J7RgtC/0YDQsNCy0LjRgtC10LvRjCDQv9C40YHRjNC80LA=?= <sender@xn--e1aybc.xn--p1ai>
To: s@example.com
Subject: =?UTF-8?b?dGVzdCDRgSDRgNGD0YHRgdC60LjQvNC4INGB0LjQvNCy0L7Qu9Cw0LzQuA==?=
Content-Type: text/plain; charset=UTF-8
Content-Type-Encoding: base64

VGVzdA==`,
			errText: "",
			isErr:   false,
		},
		{
			name: "В получаете один русский домен",
			mail: &Mail{
				From:    "Отправитель письма <sender@тест.рф>",
				To:      []string{"sender@тест.рф"},
				Subject: "test с русскими символами",
				Message: "Test",
			},
			expectedOut: `From: =?UTF-8?b?0J7RgtC/0YDQsNCy0LjRgtC10LvRjCDQv9C40YHRjNC80LA=?= <sender@xn--e1aybc.xn--p1ai>
To: sender@xn--e1aybc.xn--p1ai
Subject: =?UTF-8?b?dGVzdCDRgSDRgNGD0YHRgdC60LjQvNC4INGB0LjQvNCy0L7Qu9Cw0LzQuA==?=
Content-Type: text/plain; charset=UTF-8
Content-Type-Encoding: base64

VGVzdA==`,
			errText: "",
			isErr:   false,
		},
		{
			name: "В получаете несколько русских доменов",
			mail: &Mail{
				From:    "Отправитель письма <sender@тест.рф>",
				To:      []string{"sender@тест.рф", "test@тест.рф"},
				Subject: "test с русскими символами",
				Message: "Test",
			},
			expectedOut: `From: =?UTF-8?b?0J7RgtC/0YDQsNCy0LjRgtC10LvRjCDQv9C40YHRjNC80LA=?= <sender@xn--e1aybc.xn--p1ai>
To: sender@xn--e1aybc.xn--p1ai, test@xn--e1aybc.xn--p1ai
Subject: =?UTF-8?b?dGVzdCDRgSDRgNGD0YHRgdC60LjQvNC4INGB0LjQvNCy0L7Qu9Cw0LzQuA==?=
Content-Type: text/plain; charset=UTF-8
Content-Type-Encoding: base64

VGVzdA==`,
			errText: "",
			isErr:   false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			out, err := buildMailBody(testCase.mail)

			assert.Equal(t, testCase.expectedOut, out)

			if testCase.isErr {
				assert.Error(t, err)
				assert.Equal(t, testCase.errText, err.Error())
			} else {
				assert.NoError(t, err)
			}

		})
	}
}

func TestMailTesterSendMail(t *testing.T) {
	testMail := &Mail{
		From:    "test@test.ru",
		To:      []string{"test@test.ru"},
		Subject: "Test",
		Message: "Message body",
	}

	mailBody, _ := buildMailBody(testMail)
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
	testMail := &Mail{
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
	testMail := &Mail{
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
		t.Fatal("таймаут")
	}
}
