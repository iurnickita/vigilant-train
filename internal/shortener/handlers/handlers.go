package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/iurnickita/vigilant-train/internal/shortener/handlers/config"
	"github.com/iurnickita/vigilant-train/internal/shortener/logger"
	"github.com/iurnickita/vigilant-train/internal/shortener/service"
	"go.uber.org/zap"
)

func Serve(cfg config.Config, shortener Shortener, zaplog *zap.Logger) error {
	h := newHandlers(shortener, cfg.BaseAddr, zaplog)
	router, _ := h.newRouter()

	srv := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: router,
	}

	return srv.ListenAndServe()
}

type Shortener interface {
	GetShortener(req *service.GetShortenerRequest) (*service.GetShortenerResponse, error)
	SetShortener(req *service.SetShortenerRequest) (*service.SetShortenerResponse, error)
}

type handlers struct {
	shortener Shortener
	baseaddr  string
	zaplog    *zap.Logger
}

func newHandlers(shortener Shortener, baseaddr string, zaplog *zap.Logger) *handlers {
	return &handlers{
		shortener: shortener,
		baseaddr:  baseaddr,
		zaplog:    zaplog,
	}
}

func (h *handlers) newRouter() (*http.ServeMux, *chi.Mux) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{code}", logger.RequestLogMdlw(h.GetShortener, h.zaplog))
	mux.HandleFunc("POST /", logger.RequestLogMdlw(h.SetShortener, h.zaplog))
	mux.HandleFunc("POST /api/shorten", logger.RequestLogMdlw(h.SetShortenerJSON, h.zaplog))

	chi := chi.NewRouter() // dummy

	return mux, chi
}

func (h *handlers) GetShortener(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	resp, err := h.shortener.GetShortener(&service.GetShortenerRequest{
		Code: code,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, resp.URL, http.StatusTemporaryRedirect)
}

func (h *handlers) SetShortener(w http.ResponseWriter, r *http.Request) {
	url, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := h.shortener.SetShortener(&service.SetShortenerRequest{
		URL: string(url),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	io.WriteString(w, fmt.Sprintf("http://%s/%s", h.baseaddr, resp.Code))
}

type RawURLJSON struct {
	URL string `json:"url"`
}

type ShortURLJSON struct {
	Result string `json:"result"`
}

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

	resp, err := h.shortener.SetShortener(&service.SetShortenerRequest{
		URL: rawURL.URL,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var shortURL ShortURLJSON
	shortURL.Result = fmt.Sprintf("http://%s/%s", h.baseaddr, resp.Code)
	respJSON, err := json.Marshal(shortURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(respJSON)
}
