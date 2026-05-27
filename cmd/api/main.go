package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/artem/project/internal/config"
	"github.com/artem/project/internal/httpapi"
	"github.com/artem/project/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg := config.Load()
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("Не удалось создать пул подключений к PostgreSQL", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		logger.Error("PostgreSQL не отвечает", "error", err)
		os.Exit(1)
	}

	logger.Info("Подключился к PostgreSQL")

	repo := repository.New(pool)
	handler := httpapi.New(repo, logger)

	logger.Info("Сервер начал работу", "addr", cfg.HTTPAddr)
	if err := http.ListenAndServe(cfg.HTTPAddr, handler.Routes()); err != nil {
		logger.Error("HTTP-сервер остановился с ошибкой", "error", err)
		os.Exit(1)
	}
}
