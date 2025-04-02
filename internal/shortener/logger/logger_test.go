package logger

import (
	"net/http"
	"testing"

	"github.com/iurnickita/vigilant-train/internal/shortener/config"
)

func TestLogger_NewZapLog(t *testing.T) {
	var cfg config.Config
	cfg.Logger.LogLevel = "1"

	_, err := NewZapLog(cfg.Logger)
	if err != nil {
		t.Errorf("NewZapLog error: %s", err.Error())
	}
}

func ExampleLogger() {
	var cfg config.Config
	cfg.Logger.LogLevel = "1"

	zaplog, err := NewZapLog(cfg.Logger)
	if err != nil {
		//return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{code}", RequestLogMdlw(dummyHandler, zaplog))
}

func dummyHandler(w http.ResponseWriter, r *http.Request) {}
