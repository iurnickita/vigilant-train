// Пакет handlers. Обработчики http
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/signal"
	"strings"
	"syscall"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/iurnickita/vigilant-train/internal/shortener/auth"
	"github.com/iurnickita/vigilant-train/internal/shortener/gzip"
	"github.com/iurnickita/vigilant-train/internal/shortener/handlers/config"
	"github.com/iurnickita/vigilant-train/internal/shortener/logger"
	"github.com/iurnickita/vigilant-train/internal/shortener/model"
	"github.com/iurnickita/vigilant-train/internal/shortener/repository"
	"github.com/iurnickita/vigilant-train/internal/shortener/service"
)

// Serve - запуск сервера
func Serve(cfg config.Config, shortener service.Service, zaplog *zap.Logger) error {
	h := newHandlers(cfg, shortener, zaplog)
	router, _ := h.newRouter()

	srv := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: router,
	}

	// graceful shutdown
	idleConnsClosed := make(chan struct{})
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer stop()
	go func() {
		<-ctx.Done()
		if err := srv.Shutdown(context.Background()); err != nil {
			zaplog.Info("HTTP server Shutdown error",
				zap.String("error", err.Error()))
		}
		// сообщаем основному потоку,
		// что все сетевые соединения обработаны и закрыты
		close(idleConnsClosed)
	}()

	var serveErr error
	if cfg.EnableHTTPS {
		serveErr = listenAndServeTLS(srv)
	} else {
		serveErr = srv.ListenAndServe()
	}
	if serveErr != http.ErrServerClosed {
		return serveErr
	}

	// ждём завершения процедуры graceful shutdown
	<-idleConnsClosed
	return nil
}

// handlers. Обработчики http
type handlers struct {
	config    config.Config
	shortener service.Service
	zaplog    *zap.Logger
}

func newHandlers(config config.Config, shortener service.Service, zaplog *zap.Logger) *handlers {
	return &handlers{
		config:    config,
		shortener: shortener,
		zaplog:    zaplog,
	}
}

// newRouter формирует mux роутер
// здесь обработчики привязываются к эндпоинтам и добавляются все необходимые middleware
func (h *handlers) newRouter() (*http.ServeMux, *chi.Mux) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{code}", logger.RequestLogMdlw(gzip.GzipMiddleware(h.GetShortener), h.zaplog))
	mux.HandleFunc("POST /", logger.RequestLogMdlw(gzip.GzipMiddleware(auth.AuthMiddleware(h.SetShortener)), h.zaplog))
	mux.HandleFunc("POST /api/shorten", logger.RequestLogMdlw(gzip.GzipMiddleware(auth.AuthMiddleware(h.SetShortenerJSON)), h.zaplog))
	mux.HandleFunc("POST /api/shorten/batch", logger.RequestLogMdlw(gzip.GzipMiddleware(auth.AuthMiddleware(h.SetShortenerJSONBatch)), h.zaplog))
	mux.HandleFunc("GET /ping", logger.RequestLogMdlw(h.Ping, h.zaplog))
	mux.HandleFunc("GET /api/user/urls", logger.RequestLogMdlw(gzip.GzipMiddleware(auth.AuthMiddleware(h.GetUserURLs)), h.zaplog))
	mux.HandleFunc("DELETE /api/user/urls", logger.RequestLogMdlw(gzip.GzipMiddleware(auth.AuthMiddleware(h.DeleteShortenerBatch)), h.zaplog))

	chi := chi.NewRouter() // dummy

	return mux, chi
}

// Обработчик GetShortener перенаправляет по короткой ссылке
func (h *handlers) GetShortener(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	resp, err := h.shortener.GetShortener(code)
	if err != nil {
		if errors.Is(err, repository.ErrGetShortenerGone) {
			http.Error(w, err.Error(), http.StatusGone)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	http.Redirect(w, r, resp.Data.URL, http.StatusTemporaryRedirect)
}

// Обработчик SetShortener создает короткую ссылку
func (h *handlers) SetShortener(w http.ResponseWriter, r *http.Request) {
	url, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userCode := r.Header.Get(auth.UserCodeKey)

	resp, err := h.shortener.SetShortener(model.Shortener{
		Data: model.ShortenerData{URL: string(url), User: userCode},
	})
	if err != nil {
		if resp.Key.Code != "" {
			w.WriteHeader(http.StatusConflict)
			io.WriteString(w, fmt.Sprintf("http://%s/%s", h.config.BaseAddr, resp.Key.Code))
			return
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
	io.WriteString(w, fmt.Sprintf("http://%s/%s", h.config.BaseAddr, resp.Key.Code))
}

// Обработчик SetShortenerJSON: JSON запроса с исходным URL
type RawURLJSON struct {
	URL string `json:"url"`
}

// Обработчик SetShortenerJSON: JSON ответа с короткой ссылкой
type ShortURLJSON struct {
	Result string `json:"result"`
}

// Обработчик SetShortenerJSON создает короткую ссылку
func (h *handlers) SetShortenerJSON(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var rawURL RawURLJSON
	err = json.Unmarshal(buf.Bytes(), &rawURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userCode := r.Header.Get(auth.UserCodeKey)

	resp, err := h.shortener.SetShortener(model.Shortener{
		Data: model.ShortenerData{URL: rawURL.URL, User: userCode},
	})

	httpStatus := http.StatusCreated
	if err != nil {
		if resp.Key.Code != "" {
			httpStatus = http.StatusConflict
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	var shortURL ShortURLJSON
	shortURL.Result = fmt.Sprintf("http://%s/%s", h.config.BaseAddr, resp.Key.Code)
	respJSON, err := json.Marshal(shortURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	w.Write(respJSON)
}

// Обработчик SetShortenerJSONBatch: JSON запроса с исходным URL
type SetShortenerJSONBatchRRow struct {
	ID     string `json:"correlation_id"`
	RawURL string `json:"original_url"`
}

// Обработчик SetShortenerJSONBatch: JSON запроса с исходным URL (набор)
type SetShortenerJSONBatchR []SetShortenerJSONBatchRRow

// Обработчик SetShortenerJSONBatch: JSON ответа с коротким URL
type SetShortenerJSONBatchWRow struct {
	ID       string `json:"correlation_id"`
	ShortURL string `json:"short_url"`
}

// Обработчик SetShortenerJSONBatch: JSON ответа с коротким URL (набор)
type SetShortenerJSONBatchW []SetShortenerJSONBatchWRow

// Обработчик SetShortenerJSONBatch создает короткую ссылку для набора URL
func (h *handlers) SetShortenerJSONBatch(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var request SetShortenerJSONBatchR
	err = json.Unmarshal(buf.Bytes(), &request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userCode := r.Header.Get(auth.UserCodeKey)

	var requestService []model.Shortener
	for _, row := range request {
		requestService = append(requestService, model.Shortener{Data: model.ShortenerData{URL: row.RawURL, User: userCode}})
	}

	responseService, err := h.shortener.SetShortenerBatch(requestService)

	httpStatus := http.StatusCreated
	if err != nil {
		if len(responseService) > 0 {
			httpStatus = http.StatusConflict
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	var response SetShortenerJSONBatchW
	for _, respRow := range responseService {
		var id string
		id = ""
		for _, reqRow := range request { // вместо этого можно было пропустить ID сквозь SetShortenerBatch, но я спешу
			if reqRow.RawURL == respRow.Data.URL {
				id = reqRow.ID
				break
			}
		}
		if id != "" {
			shortURL := fmt.Sprintf("http://%s/%s", h.config.BaseAddr, respRow.Key.Code)
			response = append(response, SetShortenerJSONBatchWRow{ID: id, ShortURL: shortURL})
		}
	}

	if len(response) > 0 {
		responseJSON, err := json.Marshal(response)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(httpStatus)
		w.Write(responseJSON)
	}

}

// Обработчик Ping проверяет состояние сервера
func (h *handlers) Ping(w http.ResponseWriter, r *http.Request) {
	err := h.shortener.Ping()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Обработчик GetUserURLs: JSON ответа с коротким URL
type GetUserURLsJSON struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// Обработчик GetUserURLs возвращает все ссылки, добавленные пользователем
func (h *handlers) GetUserURLs(w http.ResponseWriter, r *http.Request) {
	userCode := r.Header.Get(auth.UserCodeKey)

	batch, err := h.shortener.GetShortnerBatchUser(userCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var response []GetUserURLsJSON
	for _, row := range batch {
		shortURL := fmt.Sprintf("http://%s/%s", h.config.BaseAddr, row.Key.Code)
		response = append(response, GetUserURLsJSON{ShortURL: shortURL, OriginalURL: row.Data.URL})
	}

	if len(response) > 0 {
		responseJSON, err := json.Marshal(response)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(responseJSON)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

// Обработчик DeleteShortenerBatch удаление набора ссылок
func (h *handlers) DeleteShortenerBatch(w http.ResponseWriter, r *http.Request) {
	// получение id пользователя
	userCode := r.Header.Get(auth.UserCodeKey)

	// Чтение body
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var codeArr []string
	err = json.Unmarshal(buf.Bytes(), &codeArr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Конвертация
	s := make([]model.Shortener, 0, len(codeArr))
	for _, code := range codeArr {
		s = append(s, model.Shortener{Key: model.ShortenerKey{Code: code}, Data: model.ShortenerData{User: userCode}})
	}

	// Вызов метода сервиса
	go func() {
		h.shortener.DeleteShortenerBatch(s)
	}()

	w.WriteHeader(http.StatusAccepted)
}

// GetStats возвращает статистические данные
func (h *handlers) GetStats(w http.ResponseWriter, r *http.Request) {
	//Доверенная подсеть
	if h.config.TrustedSubnet == "" {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	// смотрим заголовок запроса X-Real-IP
	ipStr := r.Header.Get("X-Real-IP")
	if ipStr == "" {
		// если заголовок X-Real-IP пуст, пробуем X-Forwarded-For
		// этот заголовок содержит адреса отправителя и промежуточных прокси
		// в виде 203.0.113.195, 70.41.3.18, 150.172.238.178
		ips := r.Header.Get("X-Forwarded-For")
		// разделяем цепочку адресов
		ipStrs := strings.Split(ips, ",")
		// интересует только первый
		ipStr = ipStrs[0]
	}
	if !(len(ipStr) > len(h.config.TrustedSubnet)) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	if h.config.TrustedSubnet != ipStr[:len(h.config.TrustedSubnet)] {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	stats, err := h.shortener.GetStats(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	responseJSON, err := json.Marshal(stats)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseJSON)
}
