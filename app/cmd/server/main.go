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

	"cloudops-platform/internal/api"
	"cloudops-platform/internal/auth"
	"cloudops-platform/internal/checker"
	"cloudops-platform/internal/config"
	"cloudops-platform/internal/metrics"
	"cloudops-platform/internal/store"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(log)
	if err := run(log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg := config.Load()

	for _, t := range append(append([]string{}, cfg.APITokens...), cfg.APIReadTokens...) {
		if len(t) < 16 {
			return errors.New("every API token must be at least 16 characters")
		}
	}

	// Optional per-user auth. Only assign when configured, so the interface stays nil otherwise.
	var verifier api.TokenVerifier
	if cfg.OIDCIssuer != "" {
		if cfg.OIDCAudience == "" || cfg.OIDCJWKSURL == "" {
			return errors.New("OIDC_AUDIENCE and OIDC_JWKS_URL are required when OIDC_ISSUER is set")
		}
		verifier = auth.NewVerifier(auth.Config{
			Issuer:   cfg.OIDCIssuer,
			Audience: cfg.OIDCAudience,
			JWKSURL:  cfg.OIDCJWKSURL,
		})
	}

	if len(cfg.APITokens) == 0 && len(cfg.APIReadTokens) == 0 && verifier == nil {
		if !cfg.AllowNoAuth {
			return errors.New("set API_TOKENS, API_READ_TOKENS and/or OIDC_ISSUER (or ALLOW_NO_AUTH=true for local development only)")
		}
		log.Warn("authentication is DISABLED (ALLOW_NO_AUTH=true); never use this outside local development")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer st.Close()

	m := metrics.New()
	srv := api.New(st, m, log, cfg.AllowPrivateTargets, api.AuthOptions{
		Tokens:     cfg.APITokens,
		ReadTokens: cfg.APIReadTokens,
		Verifier:   verifier,
		ReadRole:   cfg.OIDCReadRole,
		WriteRole:  cfg.OIDCWriteRole,
	})
	chk := checker.New(st, m, log, cfg)
	go chk.Run(ctx)

	httpSrv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", cfg.HTTPAddr)
		errCh <- httpSrv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		log.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return httpSrv.Shutdown(shutdownCtx)
}
