package logger

import (
	"net/http"
	"strconv"
	"time"

	"github.com/iurnickita/vigilant-train/internal/shortener/logger/config"
	"go.uber.org/zap"
)

// NewZapLog создает объект zap-логера
func NewZapLog(cfg config.Config) (*zap.Logger, error) {
	// преобразуем текстовый уровень логирования в zap.AtomicLevel
	lvl, err := zap.ParseAtomicLevel(cfg.LogLevel)
	if err != nil {
		return nil, err
	}
	// создаём новую конфигурацию логера
	zapcfg := zap.NewProductionConfig()
	// устанавливаем уровень
	zapcfg.Level = lvl
	// создаём логер на основе конфигурации
	zl, err := zapcfg.Build()
	if err != nil {
		return nil, err
	}
	//
	return zl, nil
}

// RequestLogMdlw middleware-логер для входящих HTTP-запросов.
func RequestLogMdlw(h http.HandlerFunc, zaplog *zap.Logger) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		zaplog.Info("got incoming HTTP request",
			zap.String("path", r.URL.Path),
			zap.String("method", r.Method),
		)

		wl := NewResponseWriterLogger(w)

		handlerStart := time.Now()
		h(wl, r)
		handlerDuration := time.Since(handlerStart)

		zaplog.Info("send HTTP response",
			zap.String("code", strconv.Itoa(wl.statusCode)),
			zap.String("length", strconv.Itoa(wl.length)),
			zap.String("duration", handlerDuration.String()),
		)

	})
}

type responseWriterLogger struct {
	http.ResponseWriter
	statusCode int
	length     int
}

// NewResponseWriterLogger оборачивает http.ResponseWriter дополнительным слоем логгирования
func NewResponseWriterLogger(w http.ResponseWriter) *responseWriterLogger {
	return &responseWriterLogger{w, http.StatusOK, 0}
}

// WriteHeader переопределение
func (wl *responseWriterLogger) WriteHeader(code int) {
	wl.statusCode = code
	wl.ResponseWriter.WriteHeader(code)
}

// Write переопределение
func (wl *responseWriterLogger) Write(b []byte) (n int, err error) {
	n, err = wl.ResponseWriter.Write(b)
	wl.length += n
	return
}
