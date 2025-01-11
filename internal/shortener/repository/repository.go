package repository

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	_ "github.com/jackc/pgx"

	"github.com/iurnickita/vigilant-train/internal/shortener/repository/config"
)

// Интерфейс

type Repository interface {
	GetShortener(req *GetShortenerRequest) (*GetShortenerResponse, error)
	SetShortener(req *SetShortenerRequest)
	Ping() error
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
	case config.StoreTypeDB:
		if cfg.DbDsn != "" {
			return NewStoreDB(cfg)
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

func (s *StoreVar) Ping() error {
	return nil
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

func NewStoreFile(cfg config.Config) (*StoreFile, error) {
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

func (s *StoreFile) Ping() error {
	return nil
}

// Реализация с хранением в базе данных

type StoreDB struct {
	mux       *sync.Mutex
	shortener map[string]string
	database  *sql.DB
}

func NewStoreDB(cfg config.Config) (*StoreDB, error) {
	ps := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		`localhost`, `shortener`, `shortener`, `shortener`)

	db, err := sql.Open("pgx", ps)
	if err != nil {
		return nil, err
	}

	return &StoreDB{
		mux:       &sync.Mutex{},
		shortener: make(map[string]string),
		database:  db,
	}, nil
}

func (s *StoreDB) GetShortener(req *GetShortenerRequest) (*GetShortenerResponse, error) {
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

func (s *StoreDB) SetShortener(req *SetShortenerRequest) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.shortener[req.Code] = req.URL
}

func (s *StoreDB) Ping() error {
	return s.database.Ping()
}
