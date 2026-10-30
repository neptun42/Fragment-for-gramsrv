package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"lunax/internal/config"
	"lunax/internal/domain"
	"lunax/internal/httpapi"
	"lunax/internal/network"
	_ "lunax/internal/network/gramsrv"
	_ "lunax/internal/network/webhook"
	"lunax/internal/secret"
	"lunax/internal/service"
	"lunax/internal/store/postgres"
)

// основа мейна: собираем зависимости и запускаем http
func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("lunax stopped", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	box, err := secret.NewBox(cfg.SecretKey)
	if err != nil {
		return err
	}
	db, err := postgres.Open(ctx, cfg.DatabaseURL, box)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.Migrate(ctx); err != nil {
		return err
	}

	registry := network.NewRegistry(db)
	servers := service.NewServers(db, registry, cfg.AutoApprove)
	market := service.NewMarket(db, log)

	if cfg.ServersFile != "" {
		if err := loadServersFile(ctx, cfg.ServersFile, servers, log); err != nil {
			return err
		}
	}

	api := httpapi.New(httpapi.Deps{
		Auth:       service.NewAuth(db, registry, domain.Cents(cfg.WelcomeBonus*100)),
		Market:     market,
		Wallet:     service.NewWallet(db, registry, log),
		Servers:    servers,
		AdminToken: cfg.AdminToken,
		CORSOrigin: cfg.CORSOrigin,
		Log:        log,
	})

	go market.RunSettler(ctx, 10*time.Second)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           withStatic(api, cfg.StaticDir),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shut, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shut)
	}()

	log.Info("lunax listening", "addr", cfg.Addr, "kinds", network.Kinds())
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
