package repository

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/iurnickita/vigilant-train/internal/shortener/repository/config"
)

// Интерфейс

type Repository interface {
	GetShortener(req *GetShortenerRequest) (*GetShortenerResponse, error)
	SetShortener(req *SetShortenerRequest)
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

type SetShortenerRequest struct {
	Code string
	URL  string
}

func NewStore(cfg config.Config) (Repository, error) {
	switch cfg.StoreType {
	case config.StoreTypeFile:
		if cfg.Filename != "" {
			return NewStoreFile(cfg)
		}
	}
	return NewStoreVar(cfg)
}

// Реализация с хранением в переменной

type StoreVar struct {
	mux       *sync.Mutex
	shortener map[string]string
}

func NewStoreVar(cfg config.Config) (*StoreVar, error) {
	return &StoreVar{
		mux:       &sync.Mutex{},
		shortener: make(map[string]string),
	}, nil
}

func (s *StoreVar) GetShortener(req *GetShortenerRequest) (*GetShortenerResponse, error) {
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

func (s *StoreVar) SetShortener(req *SetShortenerRequest) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.shortener[req.Code] = req.URL
}

// Реализация с хранением в файле

type StoreFile struct {
	mux       *sync.Mutex
	shortener map[string]string
	writer    *bufio.Writer
}

type FileJSON struct {
	Code string `json:"code"`
	URL  string `json:"url"`
}

func NewStoreFile(cfg config.Config) (Repository, error) {
	file, err := os.OpenFile(cfg.Filename, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	// Наполнение мапы из файла
	scanner := bufio.NewScanner(file)
	var fileJSON FileJSON
	shortener := map[string]string{}
	for scanner.Scan() {
		if err := json.Unmarshal(scanner.Bytes(), &fileJSON); err == nil && fileJSON.Code != "" {
			shortener[fileJSON.Code] = fileJSON.URL
		}
	}

	return &StoreFile{
		mux:       &sync.Mutex{},
		shortener: shortener,
		writer:    bufio.NewWriter(file),
	}, nil
}

func (s *StoreFile) GetShortener(req *GetShortenerRequest) (*GetShortenerResponse, error) {
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

func (s *StoreFile) SetShortener(req *SetShortenerRequest) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.shortener[req.Code] = req.URL

	fileJSON := FileJSON{Code: req.Code, URL: req.URL}
	data, err := json.Marshal(&fileJSON)
	if err != nil {
		return
	}

	// записываем в буфер
	if _, err := s.writer.Write(data); err != nil {
		return
	}

	// добавляем перенос строки
	if err := s.writer.WriteByte('\n'); err != nil {
		return
	}

	// записываем буфер в файл
	s.writer.Flush()

}
