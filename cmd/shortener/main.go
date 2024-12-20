package main

import (
	"log"

	"github.com/iurnickita/vigilant-train/internal/shortener/config"
	"github.com/iurnickita/vigilant-train/internal/shortener/handlers"
	"github.com/iurnickita/vigilant-train/internal/shortener/logger"
	"github.com/iurnickita/vigilant-train/internal/shortener/repository"
	"github.com/iurnickita/vigilant-train/internal/shortener/service"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg := config.GetConfig()

	zaplog, err := logger.NewZapLog(cfg.Logger)
	if err != nil {
		return err
	}

	store := repository.NewStore()
	shortenerService := service.NewShortener(store)

	return handlers.Serve(cfg.Handlers, shortenerService, zaplog)
}

// curl -v -X POST -d https://practicum.yandex.ru/ http://localhost:8080/
