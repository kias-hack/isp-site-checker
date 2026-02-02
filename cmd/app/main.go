package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kias-hack/isp-site-checker/internal/checker"
	"github.com/kias-hack/isp-site-checker/internal/config"
	"github.com/kias-hack/isp-site-checker/internal/notify"
)

func main() {
	cfg := config.LoadConfig()

	if cfg.DebugMode {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	chk := checker.NewChecker(cfg, notify.NewSender(cfg))

	if err := chk.Start(); err != nil {
		panic(err)
	}

	sig := make(chan os.Signal, 1)

	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	<-sig

	slog.Info("получил сигнал на завершение, завершаюсь")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := chk.Stop(ctx); err != nil {
		slog.Info("при завершении произошла ошибка", "err", err)
	}
}
