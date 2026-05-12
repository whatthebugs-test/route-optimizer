package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/kestrelfreight/route-optimizer/internal/api"
	"github.com/kestrelfreight/route-optimizer/internal/cache"
	"github.com/kestrelfreight/route-optimizer/internal/store"
)

func main() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	if os.Getenv("LOG_PRETTY") != "" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := store.New(ctx, mustEnv("DATABASE_URL"))
	if err != nil {
		log.Fatal().Err(err).Msg("connect database")
	}
	defer db.Close()

	rc, err := cache.New(envOr("REDIS_URL", "redis://localhost:6379/1"))
	if err != nil {
		log.Fatal().Err(err).Msg("configure redis")
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), requestLog())
	(&api.Server{Store: db, Cache: rc}).Routes(r, []byte(mustEnv("JWT_SECRET")))

	srv := &http.Server{
		Addr:              ":" + envOr("PORT", "8090"),
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	go func() {
		log.Info().Str("addr", srv.Addr).Msg("route-optimizer listening")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("listen")
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("shutdown")
	}
}

func requestLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		if c.Request.URL.Path == "/healthz" {
			return
		}
		log.Info().Str("method", c.Request.Method).Str("path", c.FullPath()).
			Int("status", c.Writer.Status()).Dur("took", time.Since(start)).Msg("request")
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatal().Str("var", key).Msg("missing required environment variable")
	}
	return v
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
