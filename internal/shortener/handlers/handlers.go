package handlers

import (
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/iurnickita/vigilant-train/internal/shortener/handlers/config"
	"github.com/iurnickita/vigilant-train/internal/shortener/service"
)

func Serve(cfg config.Config, shortener Shortener) error {
	h := newHandlers(shortener, cfg.BaseAddr)
	router, _ := newRouter(h)

	srv := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: router,
	}

	return srv.ListenAndServe()
}

func newRouter(h *handlers) (*http.ServeMux, *chi.Mux) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{code}", h.GetShortener)
	mux.HandleFunc("POST /", h.SetShortener)

	chi := chi.NewRouter() // dummy

	return mux, chi
}

type Shortener interface {
	GetShortener(req *service.GetShortenerRequest) (*service.GetShortenerResponse, error)
	SetShortener(req *service.SetShortenerRequest) (*service.SetShortenerResponse, error)
}

type handlers struct {
	shortener Shortener
	baseaddr  string
}

func newHandlers(shortener Shortener, baseaddr string) *handlers {
	return &handlers{
		shortener: shortener,
		baseaddr:  baseaddr,
	}
}

func (h *handlers) GetShortener(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	resp, err := h.shortener.GetShortener(&service.GetShortenerRequest{
		Code: code,
	})
	if err != nil {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, resp.URL, http.StatusTemporaryRedirect)
}

func (h *handlers) SetShortener(w http.ResponseWriter, r *http.Request) {
	url, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	resp, err := h.shortener.SetShortener(&service.SetShortenerRequest{
		URL: string(url),
	})
	if err != nil {
		http.Error(w, "", http.StatusBadRequest)
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Add("Content-Type", "text/plain")
	io.WriteString(w, fmt.Sprintf("http://%s/%s", h.baseaddr, resp.Code))
}
