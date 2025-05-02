package logger

import (
	"net/http"

	"github.com/iurnickita/vigilant-train/internal/shortener/config"
)

func ExampleLogger() {
	var cfg config.Config
	cfg.Logger.LogLevel = "info"

	zaplog, err := NewZapLog(cfg.Logger)
	if err != nil {
		//return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{code}", RequestLogMdlw(dummyHandler, zaplog))
}

func dummyHandler(w http.ResponseWriter, r *http.Request) {}
