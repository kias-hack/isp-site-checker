package main

import (
	"context"
	"flag"
	"log/slog"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kias-hack/isp-site-checker/internal/checker"
	"github.com/kias-hack/isp-site-checker/internal/config"
	"github.com/kias-hack/isp-site-checker/internal/isp"
	"github.com/kias-hack/isp-site-checker/internal/notify"
	"github.com/kias-hack/isp-site-checker/internal/util"
)

func main() {
	var configPath string
	var debugMode bool

	flag.StringVar(&configPath, "config", "", "")
	flag.BoolVar(&debugMode, "debug", false, "")

	flag.Parse()

	cfg, err := config.LoadConfig(configPath, util.GetUsernameAndHostByEmailAddress)
	if err != nil {
		slog.Error("config error", "err", err)
		os.Exit(1)
	}

	if debugMode {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	if flag.Arg(0) == "sendmail" {
		slog.SetLogLoggerLevel(slog.LevelDebug)

		email := &util.Mail{
			From:    cfg.EMail.From,
			To:      cfg.EMail.To,
			Subject: cfg.EMail.Subject,
			Message: "test",
		}
		body, err := util.BuildMailBody(email)
		if err != nil {
			slog.Error("failed build mail body", "err", err)
			os.Exit(1)
		}

		address, err := mail.ParseAddress(cfg.EMail.From)
		if err != nil {
			slog.Error("failed parse from address", "err", err)
			os.Exit(1)
		}
		auth := smtp.PlainAuth("", cfg.SMTP.Username, cfg.SMTP.Password, cfg.SMTP.Host)

		slog.Debug("send email with params", "addr", net.JoinHostPort(cfg.SMTP.Host, cfg.SMTP.Port),
			"auth", auth, "host", address.Address, "to", []string{flag.Arg(1)}, "message", []byte(body))
		err = util.SendMail(net.JoinHostPort(cfg.SMTP.Host, cfg.SMTP.Port), auth, address.Address, cfg.EMail.To, []byte(body), !cfg.SMTP.UseTLS)
		if err != nil {
			slog.Error("failed send email", "err", err)
			os.Exit(1)
		}

		os.Exit(0)
	}

	sender := notify.NewMailSender(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.Username, cfg.SMTP.Password, func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		return util.SendMail(addr, a, from, to, msg, !cfg.SMTP.UseTLS)
	})
	webDomainsFunc := func() ([]*isp.WebDomain, error) {
		return isp.GetWebDomains(cfg.MgrCtlPath)
	}

	chk := checker.NewChecker(cfg, notify.NewNotifier(cfg, sender), webDomainsFunc)

	if err := chk.Start(); err != nil {
		slog.Error("failed to start application", "err", err)
		os.Exit(1)
	}

	sig := make(chan os.Signal, 1)

	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	<-sig

	slog.Info("received shutdown signal, exiting")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := chk.Stop(ctx); err != nil {
		slog.Info("error during shutdown", "err", err)
		os.Exit(1)
	}
}
