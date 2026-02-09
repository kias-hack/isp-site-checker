package util

import (
	"testing"

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
			name: "empty sender",
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
			name: "zero recipients",
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
			name: "empty recipient",
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
			name: "empty subject",
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
			name: "empty body",
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
			name: "successful build",
			mail: &Mail{
				From:    "me@example.com",
				To:      []string{"s@example.com"},
				Subject: "test",
				Message: "Test",
			},
			expectedOut: "From: me@example.com\r\nTo: s@example.com\r\nSubject: test\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\nVGVzdA==\r\n",
			errText:     "",
			isErr:       false,
		},
		{
			name: "successful build with multiple recipients",
			mail: &Mail{
				From:    "me@example.com",
				To:      []string{"s@example.com", "w@example.com"},
				Subject: "test",
				Message: "Test",
			},
			expectedOut: "From: me@example.com\r\nTo: s@example.com, w@example.com\r\nSubject: test\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\nVGVzdA==\r\n",
			errText:     "",
			isErr:       false,
		},
		{
			name: "subject with non-ASCII characters",
			mail: &Mail{
				From:    "me@example.com",
				To:      []string{"s@example.com"},
				Subject: "test с русскими символами",
				Message: "Test",
			},
			expectedOut: "From: me@example.com\r\nTo: s@example.com\r\nSubject: =?UTF-8?b?dGVzdCDRgSDRgNGD0YHRgdC60LjQvNC4INGB0LjQvNCy0L7Qu9Cw0LzQuA==?=\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\nVGVzdA==\r\n",
			errText:     "",
			isErr:       false,
		},
		{
			name: "mail with sender display name",
			mail: &Mail{
				From:    "Отправитель письма <sender@example.com>",
				To:      []string{"s@example.com"},
				Subject: "test с русскими символами",
				Message: "Test",
			},
			expectedOut: "From: =?UTF-8?b?0J7RgtC/0YDQsNCy0LjRgtC10LvRjCDQv9C40YHRjNC80LA=?= <sender@example.com>\r\nTo: s@example.com\r\nSubject: =?UTF-8?b?dGVzdCDRgSDRgNGD0YHRgdC60LjQvNC4INGB0LjQvNCy0L7Qu9Cw0LzQuA==?=\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\nVGVzdA==\r\n",
			errText:     "",
			isErr:       false,
		},
		{
			name: "sender with IDN domain",
			mail: &Mail{
				From:    "Отправитель письма <sender@тест.рф>",
				To:      []string{"s@example.com"},
				Subject: "test с русскими символами",
				Message: "Test",
			},
			expectedOut: "From: =?UTF-8?b?0J7RgtC/0YDQsNCy0LjRgtC10LvRjCDQv9C40YHRjNC80LA=?= <sender@xn--e1aybc.xn--p1ai>\r\nTo: s@example.com\r\nSubject: =?UTF-8?b?dGVzdCDRgSDRgNGD0YHRgdC60LjQvNC4INGB0LjQvNCy0L7Qu9Cw0LzQuA==?=\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\nVGVzdA==\r\n",
			errText:     "",
			isErr:       false,
		},
		{
			name: "single IDN recipient",
			mail: &Mail{
				From:    "Отправитель письма <sender@тест.рф>",
				To:      []string{"sender@тест.рф"},
				Subject: "test с русскими символами",
				Message: "Test",
			},
			expectedOut: "From: =?UTF-8?b?0J7RgtC/0YDQsNCy0LjRgtC10LvRjCDQv9C40YHRjNC80LA=?= <sender@xn--e1aybc.xn--p1ai>\r\nTo: sender@xn--e1aybc.xn--p1ai\r\nSubject: =?UTF-8?b?dGVzdCDRgSDRgNGD0YHRgdC60LjQvNC4INGB0LjQvNCy0L7Qu9Cw0LzQuA==?=\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\nVGVzdA==\r\n",
			errText:     "",
			isErr:       false,
		},
		{
			name: "multiple IDN recipients",
			mail: &Mail{
				From:    "Отправитель письма <sender@тест.рф>",
				To:      []string{"sender@тест.рф", "test@тест.рф"},
				Subject: "test с русскими символами",
				Message: "Test",
			},
			expectedOut: "From: =?UTF-8?b?0J7RgtC/0YDQsNCy0LjRgtC10LvRjCDQv9C40YHRjNC80LA=?= <sender@xn--e1aybc.xn--p1ai>\r\nTo: sender@xn--e1aybc.xn--p1ai, test@xn--e1aybc.xn--p1ai\r\nSubject: =?UTF-8?b?dGVzdCDRgSDRgNGD0YHRgdC60LjQvNC4INGB0LjQvNCy0L7Qu9Cw0LzQuA==?=\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\nVGVzdA==\r\n",
			errText:     "",
			isErr:       false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			out, err := BuildMailBody(testCase.mail)

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
