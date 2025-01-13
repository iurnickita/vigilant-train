package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/iurnickita/vigilant-train/internal/common/rand"
	"github.com/iurnickita/vigilant-train/internal/shortener/repository"
)

type Shortener struct {
	store repository.Repository
}

func NewShortener(store repository.Repository) *Shortener {
	return &Shortener{
		store: store,
	}
}

type GetShortenerRequest struct {
	Code string
}

type GetShortenerResponse struct {
	URL string
}

var (
	ErrGetShortenerInvalidRequest = errors.New("invalid get Shortener request")
	ErrRepoFailed                 = errors.New("repo failed")
)

func (s *Shortener) GetShortener(req *GetShortenerRequest) (*GetShortenerResponse, error) {

	repositoryResp, err := s.store.GetShortener(&repository.GetShortenerRequest{
		Code: req.Code,
	})
	if err != nil {
		if !errors.Is(err, repository.ErrGetShortenerNotFound) {
			return nil, fmt.Errorf("failed to fetch the Shortener result from the store: %w", err)
		}
	}

	return &GetShortenerResponse{
		URL: repositoryResp.URL,
	}, nil
}

type SetShortenerRequest struct {
	URL string
}

type SetShortenerResponse struct {
	Code string
	URL  string
}

func (s *Shortener) SetShortener(req *SetShortenerRequest) (*SetShortenerResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	code := rand.String(6)

	storeResp, err := s.store.SetShortener(ctx, &repository.SetShortenerRequest{Code: code, URL: req.URL})
	if err != nil {
		return &SetShortenerResponse{Code: storeResp.Code}, err
	}

	return &SetShortenerResponse{Code: code, URL: req.URL}, nil
}

type SetShortenerRequestBatch struct {
	Rows []SetShortenerRequest
}

type SetShortenerResponseBatch struct {
	Rows []SetShortenerResponse
}

func (s *Shortener) SetShortenerBatch(req *SetShortenerRequestBatch) (*SetShortenerResponseBatch, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// конвертация запроса ??? как это сделать компактнее ??? и без лишнего цикла
	var storeReq repository.SetShortenerRequestBatch
	for _, row := range req.Rows {
		code := rand.String(6)
		storeReq.Rows = append(storeReq.Rows, repository.SetShortenerRequest{Code: code, URL: row.URL})
	}

	storeResp, err := s.store.SetShortenerBatch(ctx, &storeReq)

	// конвертация ответа ??? как это сделать компактнее ??? и без лишнего цикла
	var resp SetShortenerResponseBatch
	for _, row := range storeResp.Rows {
		resp.Rows = append(resp.Rows, SetShortenerResponse{Code: row.Code, URL: row.URL})
	}
	return &resp, err
}

func (s *Shortener) Ping() error {
	return s.store.Ping()
}
