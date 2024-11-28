package main

import (
	"log"

	"github.com/iurnickita/vigilant-train/internal/shortener/config"
	"github.com/iurnickita/vigilant-train/internal/shortener/handlers"
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

	store := repository.NewStore()
	shortenerService := service.NewShortener(store)

	return handlers.Serve(cfg.Handlers, shortenerService)
}

// curl -v -X POST -d https://practicum.yandex.ru/ http://localhost:8080/
