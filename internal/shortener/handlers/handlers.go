package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/iurnickita/vigilant-train/internal/shortener/gzip"
	"github.com/iurnickita/vigilant-train/internal/shortener/handlers/config"
	"github.com/iurnickita/vigilant-train/internal/shortener/logger"
	"github.com/iurnickita/vigilant-train/internal/shortener/model"
	"github.com/iurnickita/vigilant-train/internal/shortener/repository"
	"github.com/iurnickita/vigilant-train/internal/shortener/service"
	"github.com/iurnickita/vigilant-train/internal/shortener/token"
	"go.uber.org/zap"
)

const cCookieUser = "shortenerUserToken"

func Serve(cfg config.Config, shortener service.Service, zaplog *zap.Logger) error {
	h := newHandlers(shortener, cfg.BaseAddr, zaplog)
	router, _ := h.newRouter()

	srv := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: router,
	}

	return srv.ListenAndServe()

}

type handlers struct {
	shortener service.Service
	baseaddr  string
	zaplog    *zap.Logger
}

func newHandlers(shortener service.Service, baseaddr string, zaplog *zap.Logger) *handlers {
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
	mux.HandleFunc("GET /api/user/urls", logger.RequestLogMdlw(gzip.GzipMiddleware(h.GetUserURLs), h.zaplog))
	mux.HandleFunc("DELETE /api/user/urls", logger.RequestLogMdlw(gzip.GzipMiddleware(h.DeleteShortenerBatch), h.zaplog))

	chi := chi.NewRouter() // dummy

	return mux, chi
}

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

func (h *handlers) SetShortener(w http.ResponseWriter, r *http.Request) {
	url, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userCode, err := h.getUserCode(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := h.shortener.SetShortener(model.Shortener{
		Data: model.ShortenerData{URL: string(url), User: userCode},
	})
	if err != nil {
		if resp.Key.Code != "" {
			w.WriteHeader(http.StatusConflict)
			io.WriteString(w, fmt.Sprintf("http://%s/%s", h.baseaddr, resp.Key.Code))
			return
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
	io.WriteString(w, fmt.Sprintf("http://%s/%s", h.baseaddr, resp.Key.Code))
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

	userCode, err := h.getUserCode(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

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
	shortURL.Result = fmt.Sprintf("http://%s/%s", h.baseaddr, resp.Key.Code)
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

	userCode, err := h.getUserCode(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

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
			shortURL := fmt.Sprintf("http://%s/%s", h.baseaddr, respRow.Key.Code)
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

func (h *handlers) getUserCode(w http.ResponseWriter, r *http.Request) (string, error) {

	// куки пользователя
	var userCode string
	tokenCookie, err := r.Cookie(cCookieUser)
	if err != nil {
		userCode = h.shortener.GetNewUserCode()
		tokenString, err := token.BuildJWTString(userCode)
		if err != nil {
			return "", err
		}
		tokenCookie := http.Cookie{
			Name:  cCookieUser,
			Value: tokenString,
		}
		http.SetCookie(w, &tokenCookie)
	} else {
		userCode, err = token.GetUserCode(tokenCookie.Value)
		if err != nil {
			return "", err
		}
	}
	return userCode, nil
}

func (h *handlers) getUserCodeReadOnly(r *http.Request) (string, error) {

	// куки пользователя
	var userCode string
	tokenCookie, err := r.Cookie(cCookieUser)
	if err != nil {
		return "", err
	} else {
		userCode, err = token.GetUserCode(tokenCookie.Value)
		if err != nil {
			return "", err
		}
	}
	return userCode, nil
}

type GetUserURLsJSON struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

func (h *handlers) GetUserURLs(w http.ResponseWriter, r *http.Request) {
	//userCode, err := h.getUserCodeReadOnly(r)
	userCode, err := h.getUserCode(w, r)
	if err != nil {
		//http.Error(w, err.Error(), http.StatusUnauthorized)
		http.Error(w, err.Error(), http.StatusNoContent)
		return
	}

	batch, err := h.shortener.GetShortnerBatchUser(userCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var response []GetUserURLsJSON
	for _, row := range batch {
		shortURL := fmt.Sprintf("http://%s/%s", h.baseaddr, row.Key.Code)
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

func (h *handlers) DeleteShortenerBatch(w http.ResponseWriter, r *http.Request) {
	// получение id пользователя
	//userCode, err := h.getUserCodeReadOnly(r)
	userCode, err := h.getUserCode(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Чтение body
	var buf bytes.Buffer
	_, err = buf.ReadFrom(r.Body)
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
