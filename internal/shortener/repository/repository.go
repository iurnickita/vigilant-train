package repository

import (
	"errors"
	"fmt"
	"sync"
)

type Repository interface {
	GetShortener(req *GetShortenerRequest) (*GetShortenerResponse, error)
	SetShortener(req *SetShortenerRequest)
}

type Store struct {
	mux       *sync.Mutex
	shortener map[string]string
}

func NewStore() *Store {
	return &Store{
		mux:       &sync.Mutex{},
		shortener: make(map[string]string),
	}
}

type GetShortenerRequest struct {
	Code string
}

type GetShortenerResponse struct {
	URL string
}

var (
	ErrGetShortenerNotFound = errors.New("data not found")
)

func newErrGetShortenerNotFound(code string) error {
	return fmt.Errorf("%w for code = %s", ErrGetShortenerNotFound, code)
}

func (s *Store) GetShortener(req *GetShortenerRequest) (*GetShortenerResponse, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	url, ok := s.shortener[req.Code]
	if !ok {
		return nil, newErrGetShortenerNotFound(req.Code)
	}
	return &GetShortenerResponse{
		URL: url,
	}, nil
}

type SetShortenerRequest struct {
	Code string
	URL  string
}

func (s *Store) SetShortener(req *SetShortenerRequest) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.shortener[req.Code] = req.URL
}
