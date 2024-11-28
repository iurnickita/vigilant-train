package service

import (
	"errors"
	"fmt"

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
	if err := getShortenerValidateRequest(req); err != nil {
		return nil, err
	}

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

func getShortenerValidateRequest(req *GetShortenerRequest) error {
	return nil
}

type SetShortenerRequest struct {
	URL string
}

type SetShortenerResponse struct {
	Code string
}

func (s *Shortener) SetShortener(req *SetShortenerRequest) (*SetShortenerResponse, error) {
	code := rand.String(6)
	s.store.SetShortener(&repository.SetShortenerRequest{Code: code, URL: req.URL})

	return &SetShortenerResponse{Code: code}, nil
}
