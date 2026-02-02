package notify

// import (
// 	"context"
// 	"net/smtp"
// 	"sync"
// 	"testing"
// 	"time"

// 	"github.com/stretchr/testify/assert"
// )

// func TestSMTPSendTimeout(t *testing.T) {
// 	sender := &SMTPSender{
// 		mu:             &sync.Mutex{},
// 		domainStatuses: make(map[string]*domainStatus),
// 	}

// 	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
// 	defer cancel()

// 	smtpSendMail = func(_ string, _ smtp.Auth, _ string, _ []string, _ []byte) error {
// 		time.Sleep(400 * time.Millisecond)

// 		return nil
// 	}

// 	sender.Error("test", "test")
// 	assert.Error(t, sender.Send(ctx))
// }
