package main

import (
	"context"
	"log/slog"
	"net/smtp"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kias-hack/isp-site-checker/internal/checker"
	"github.com/kias-hack/isp-site-checker/internal/config"
	"github.com/kias-hack/isp-site-checker/internal/isp"
	"github.com/kias-hack/isp-site-checker/internal/notify"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("при конфигурировании приложения произошла ошибка", "err", err)
		os.Exit(1)
	}

	if cfg.DebugMode {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	sender := notify.NewMailSender(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.Username, cfg.SMTP.Password, smtp.SendMail)
	webDomainsFunc := func() ([]*isp.WebDomain, error) {
		return isp.GetWebDomains(cfg.MgrCtlPath)
	}

	chk := checker.NewChecker(cfg, notify.NewNotifier(cfg, sender), webDomainsFunc)

	if err := chk.Start(); err != nil {
		slog.Error("произошла ошибка запуска приложения", "err", err)
		os.Exit(1)
	}

	sig := make(chan os.Signal, 1)

	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	<-sig

	slog.Info("получил сигнал на завершение, завершаюсь")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := chk.Stop(ctx); err != nil {
		slog.Info("при завершении произошла ошибка", "err", err)
		os.Exit(1)
	}
}
