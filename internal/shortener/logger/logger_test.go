package logger

import (
	"testing"

	"github.com/iurnickita/vigilant-train/internal/shortener/config"
)

func TestLogger_NewZapLog(t *testing.T) {
	var cfg config.Config
	cfg.Logger.LogLevel = "info"

	_, err := NewZapLog(cfg.Logger)
	if err != nil {
		t.Errorf("NewZapLog error: %s", err.Error())
	}
}
