package service

import (
	"github.com/iurnickita/vigilant-train/internal/shortener/config"
	"github.com/iurnickita/vigilant-train/internal/shortener/repository"
)

func ExampleService() {
	cfg := config.GetConfig()

	store, err := repository.NewStore(cfg.Repository)
	if err != nil {
		//return err
	}
	shortenerService := NewShortener(store)
	shortenerService.Ping()
}
