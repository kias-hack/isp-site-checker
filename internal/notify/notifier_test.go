package notify

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/kias-hack/isp-site-checker/internal/config"
	"github.com/kias-hack/isp-site-checker/internal/util"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
	"go.uber.org/mock/gomock"
)

func TestNotifierLifecycle(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	ctrl, sender := newMockSender(t)
	defer ctrl.Finish()
	sender.EXPECT().Send(gomock.Any(), gomock.Any()).Times(0)

	notifier := NewNotifier(&config.Config{
		SendInterval:   10 * time.Millisecond,
		SendTimeout:    100 * time.Millisecond,
		RepeatInterval: time.Hour,
	}, sender)

	ctx, cancel := context.WithCancel(t.Context())
	stop := make(chan struct{})
	go func() {
		notifier.Stop(ctx)
		close(stop)
	}()
	cancel()

	timer := time.NewTimer(500 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-timer.C:
		t.Fatal("timeout")
	case <-stop:
	}
}

func TestNotifierStopByContextWhileWaitingSendMail(t *testing.T) {
	wait := make(chan struct{})
	ctrl, sender := newMockSender(t)
	_ = ctrl
	defer func() {
		close(wait)
		ctrl.Finish()
	}()
	sender.EXPECT().Send(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ *util.Mail) error {
		wait <- struct{}{}
		<-wait

		return nil
	}).Times(1)
	notifier := NewNotifier(&config.Config{
		SendInterval:   1 * time.Millisecond,
		SendTimeout:    100 * time.Millisecond,
		RepeatInterval: time.Hour,
	}, sender)

	notifier.Fail("site", "message")

	timer := time.NewTimer(500 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-wait:
	case <-timer.C:
		t.Fatal("mail was not sent")
	}
	ctx, cancel := context.WithCancel(t.Context())
	stop := make(chan struct{})
	go func() {
		t.Log("ss")
		notifier.Stop(ctx)
		close(stop)
	}()
	cancel()
	timer = time.NewTimer(500 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-timer.C:
		t.Fatal("timeout")
	case <-stop:
	}
}

func TestNotifierDeduplication(t *testing.T) {
	ctrl, sender := newMockSender(t)
	defer ctrl.Finish()
	sender.EXPECT().Send(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	notifier := &notifier{
		wg:             &sync.WaitGroup{},
		timeout:        1 * time.Millisecond,
		interval:       1 * time.Millisecond,
		repeatInterval: time.Hour,
		ticker:         make(chan struct{}),
		mailSender:     sender,
		sitesMap:       make(map[string]*SiteNotification),
	}
	notifier.wg.Add(1)
	go notifier.worker()

	notifier.Fail("site", "message")
	notifier.Fail("site", "message")
	notifier.Fail("site", "message")
	notifier.Fail("site", "message")
	notifier.ticker <- struct{}{}
	notifier.ticker <- struct{}{}
	notifier.ticker <- struct{}{}

	stopNotifier(t, notifier)
}

func TestNotifierSendMessageWithError(t *testing.T) {
	const NUM_ERRORS = 2
	const EXPECTED_CALLS = 3

	ctrl, sender := newMockSender(t)
	defer ctrl.Finish()
	var errCounter int
	sender.EXPECT().Send(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ *util.Mail) error {
		if errCounter == NUM_ERRORS {
			return nil
		}
		errCounter++
		return fmt.Errorf("some error")
	}).Times(EXPECTED_CALLS)
	notifier := &notifier{
		wg:             &sync.WaitGroup{},
		timeout:        1 * time.Millisecond,
		interval:       1 * time.Millisecond,
		repeatInterval: time.Hour,
		ticker:         make(chan struct{}),
		stop:           make(chan struct{}),
		mailSender:     sender,
		sitesMap:       make(map[string]*SiteNotification),
	}
	notifier.wg.Add(1)
	go notifier.worker()

	notifier.Fail("site", "message")

	for n := range EXPECTED_CALLS + 3 {
		_ = n
		notifier.ticker <- struct{}{}
	}

	stopNotifier(t, notifier)
}

func TestNotifierGotSuccessFirst(t *testing.T) {
	ctrl, sender := newMockSender(t)
	defer ctrl.Finish()
	sender.EXPECT().Send(gomock.Any(), gomock.Any()).Times(0)
	notifier := &notifier{
		wg:             &sync.WaitGroup{},
		timeout:        1 * time.Millisecond,
		interval:       1 * time.Millisecond,
		repeatInterval: time.Hour,
		ticker:         make(chan struct{}),
		stop:           make(chan struct{}),
		mailSender:     sender,
		sitesMap:       make(map[string]*SiteNotification),
	}
	notifier.wg.Add(1)
	go notifier.worker()

	notifier.Success("site", "message")

	for n := range 3 {
		_ = n
		notifier.ticker <- struct{}{}
	}

	stopNotifier(t, notifier)
}

func TestNotifierComplexBehaviour(t *testing.T) {
	ctrl, sender := newMockSender(t)
	defer ctrl.Finish()
	sender.EXPECT().Send(gomock.Any(), gomock.Any()).Times(3)
	notifier := &notifier{
		wg:             &sync.WaitGroup{},
		timeout:        1 * time.Millisecond,
		interval:       1 * time.Millisecond,
		repeatInterval: time.Hour,
		ticker:         make(chan struct{}),
		stop:           make(chan struct{}),
		mailSender:     sender,
		sitesMap:       make(map[string]*SiteNotification),
	}
	notifier.wg.Add(1)
	go notifier.worker()

	notifier.Fail("site", "message")
	notifier.ticker <- struct{}{}
	notifier.ticker <- struct{}{}
	notifier.Success("site", "message")
	notifier.ticker <- struct{}{}
	notifier.ticker <- struct{}{}
	notifier.Fail("site", "message")
	notifier.ticker <- struct{}{}
	notifier.ticker <- struct{}{}

	stopNotifier(t, notifier)
}

func TestNotifierSendMoreThan1Message(t *testing.T) {
	ctrl, sender := newMockSender(t)
	defer ctrl.Finish()
	sender.EXPECT().Send(gomock.Any(), gomock.Any()).Times(1)
	notifier := &notifier{
		wg:             &sync.WaitGroup{},
		timeout:        1 * time.Millisecond,
		interval:       1 * time.Millisecond,
		repeatInterval: time.Hour,
		ticker:         make(chan struct{}),
		stop:           make(chan struct{}),
		mailSender:     sender,
		sitesMap:       make(map[string]*SiteNotification),
	}
	notifier.wg.Add(1)
	go notifier.worker()

	notifier.Fail("site", "message")
	notifier.Fail("site1", "message")
	notifier.Fail("site2", "message")
	notifier.ticker <- struct{}{}
	notifier.ticker <- struct{}{}
	notifier.ticker <- struct{}{}

	assert.False(t, notifier.sitesMap["site"].NeedNotify)
	assert.False(t, notifier.sitesMap["site1"].NeedNotify)
	assert.False(t, notifier.sitesMap["site2"].NeedNotify)
	stopNotifier(t, notifier)
}

// repeat notification send on persistent error
func TestNotifierRepeatSend(t *testing.T) {
	ctrl, sender := newMockSender(t)
	defer ctrl.Finish()
	sender.EXPECT().Send(gomock.Any(), gomock.Any()).Times(3)
	notifier := &notifier{
		wg:             &sync.WaitGroup{},
		timeout:        1 * time.Millisecond,
		interval:       1 * time.Millisecond,
		repeatInterval: time.Hour,
		ticker:         make(chan struct{}),
		mailSender:     sender,
		stop:           make(chan struct{}),
		sitesMap:       make(map[string]*SiteNotification),
	}
	notifier.wg.Add(1)
	go notifier.worker()

	notifier.Fail("site", "message")
	notifier.ticker <- struct{}{}
	notifier.ticker <- struct{}{}
	notifier.sitesMap["site"].LastSended = notifier.sitesMap["site"].LastSended.Add(-notifier.repeatInterval).Add(-time.Second * 3)
	notifier.ticker <- struct{}{}
	notifier.ticker <- struct{}{}
	notifier.sitesMap["site"].LastSended = notifier.sitesMap["site"].LastSended.Add(-notifier.repeatInterval).Add(-time.Second * 3)
	notifier.ticker <- struct{}{}
	notifier.ticker <- struct{}{}

	stopNotifier(t, notifier)
}

func TestNotifierDeleteOldSites(t *testing.T) {
	ctrl, sender := newMockSender(t)
	defer ctrl.Finish()
	sender.EXPECT().Send(gomock.Any(), gomock.Any()).Times(0)
	notifier := &notifier{
		wg:             &sync.WaitGroup{},
		timeout:        1 * time.Millisecond,
		interval:       1 * time.Millisecond,
		repeatInterval: time.Hour,
		ticker:         make(chan struct{}),
		mailSender:     sender,
		stop:           make(chan struct{}),
		sitesMap:       make(map[string]*SiteNotification),
	}
	notifier.wg.Add(1)
	go notifier.worker()

	notifier.Success("site", "message")
	notifier.sitesMap["site"].LastUpdated = notifier.sitesMap["site"].LastUpdated.Add(-SiteRetentionPeriod)
	notifier.ticker <- struct{}{}

	_, ok := notifier.sitesMap["site"]
	if ok {
		t.Fatal("site record was not removed after retention period")
	}
	stopNotifier(t, notifier)
}

func TestNotifierCheckNoDeletingCases(t *testing.T) {
	ctrl, sender := newMockSender(t)
	defer ctrl.Finish()
	sender.EXPECT().Send(gomock.Any(), gomock.Any()).AnyTimes()
	notifier := &notifier{
		wg:             &sync.WaitGroup{},
		timeout:        1 * time.Millisecond,
		interval:       1 * time.Millisecond,
		repeatInterval: time.Hour,
		ticker:         make(chan struct{}),
		mailSender:     sender,
		stop:           make(chan struct{}),
		sitesMap:       make(map[string]*SiteNotification),
	}
	notifier.wg.Add(1)
	go notifier.worker()

	notifier.Success("site1", "message")
	notifier.Fail("site2", "message")
	notifier.sitesMap["site1"].LastUpdated = notifier.sitesMap["site1"].LastUpdated.Add(-SiteRetentionPeriod / 2)
	notifier.sitesMap["site2"].LastUpdated = notifier.sitesMap["site2"].LastUpdated.Add(-SiteRetentionPeriod).Add(-2 * time.Hour)
	notifier.ticker <- struct{}{}

	_, ok := notifier.sitesMap["site1"]
	if !ok {
		t.Fatalf("site record %s was unexpectedly removed", "site1")
	}
	_, ok = notifier.sitesMap["site2"]
	if !ok {
		t.Fatalf("site record %s was unexpectedly removed", "site2")
	}
	stopNotifier(t, notifier)
}

func newMockSender(t *testing.T) (*gomock.Controller, *MockMailSender) {
	ctrl := gomock.NewController(t)
	sender := NewMockMailSender(ctrl)

	return ctrl, sender
}

func stopNotifier(t *testing.T, notifier *notifier) {
	timer := time.NewTimer(1 * time.Second)
	defer timer.Stop()
	wait := make(chan struct{})
	go func() {
		notifier.Stop(t.Context())
		close(wait)
	}()
	select {
	case <-timer.C:
		t.Fatal("timeout")
	case <-wait:
	}
}
