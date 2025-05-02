package service

import (
	"testing"

	"github.com/iurnickita/vigilant-train/internal/shortener/config"
	"github.com/iurnickita/vigilant-train/internal/shortener/repository"
)

func TestService_NewShortener(t *testing.T) {
	cfg := config.GetConfig()

	store, err := repository.NewStore(cfg.Repository)
	if err != nil {
		t.Error(err)
	}
	shortenerService := NewShortener(store)
	if shortenerService == nil {
		t.Errorf("NewShortener error")
	}
}
