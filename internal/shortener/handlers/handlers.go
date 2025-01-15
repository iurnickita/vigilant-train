package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/iurnickita/vigilant-train/internal/shortener/gzip"
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

type Shortener interface { // Переместить в service.go
	GetShortener(req *service.GetShortenerRequest) (*service.GetShortenerResponse, error)
	SetShortener(req *service.SetShortenerRequest) (*service.SetShortenerResponse, error)
	SetShortenerBatch(req *service.SetShortenerRequestBatch) (*service.SetShortenerResponseBatch, error)
	Ping() error
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
	mux.HandleFunc("GET /{code}", logger.RequestLogMdlw(gzip.GzipMiddleware(h.GetShortener), h.zaplog))
	mux.HandleFunc("POST /", logger.RequestLogMdlw(gzip.GzipMiddleware(h.SetShortener), h.zaplog))
	mux.HandleFunc("POST /api/shorten", logger.RequestLogMdlw(gzip.GzipMiddleware(h.SetShortenerJSON), h.zaplog))
	mux.HandleFunc("POST /api/shorten/batch", logger.RequestLogMdlw(gzip.GzipMiddleware(h.SetShortenerJSONBatch), h.zaplog))
	mux.HandleFunc("GET /ping", logger.RequestLogMdlw(h.Ping, h.zaplog))

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
		if resp.Code != "" {
			w.WriteHeader(http.StatusConflict)
			io.WriteString(w, fmt.Sprintf("http://%s/%s", h.baseaddr, resp.Code))
			return
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
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
	httpStatus := http.StatusCreated
	if err != nil {
		if resp.Code != "" {
			httpStatus = http.StatusConflict
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	var shortURL ShortURLJSON
	shortURL.Result = fmt.Sprintf("http://%s/%s", h.baseaddr, resp.Code)
	respJSON, err := json.Marshal(shortURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	w.Write(respJSON)
}

type SetShortenerJSONBatchRRow struct {
	ID     string `json:"correlation_id"`
	RawURL string `json:"original_url"`
}
type SetShortenerJSONBatchR []SetShortenerJSONBatchRRow

type SetShortenerJSONBatchWRow struct {
	ID       string `json:"correlation_id"`
	ShortURL string `json:"short_url"`
}
type SetShortenerJSONBatchW []SetShortenerJSONBatchWRow

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

	var requestService service.SetShortenerRequestBatch
	for _, row := range request {
		requestService.Rows = append(requestService.Rows, service.SetShortenerRequest{URL: row.RawURL})
	}

	responseService, err := h.shortener.SetShortenerBatch(&requestService)
	httpStatus := http.StatusCreated
	if err != nil {
		if len(responseService.Rows) > 0 {
			httpStatus = http.StatusConflict
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	var response SetShortenerJSONBatchW
	for _, respRow := range responseService.Rows {
		var id string
		id = ""
		for _, reqRow := range request { // вместо этого можно было пропустить ID сквозь SetShortenerBatch, но я спешу
			if reqRow.RawURL == respRow.URL {
				id = reqRow.ID
				break
			}
		}
		if id != "" {
			shortURL := fmt.Sprintf("http://%s/%s", h.baseaddr, respRow.Code)
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

func (h *handlers) Ping(w http.ResponseWriter, r *http.Request) {
	err := h.shortener.Ping()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
