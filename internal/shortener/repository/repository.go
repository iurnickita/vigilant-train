package repository

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/iurnickita/vigilant-train/internal/shortener/repository/config"
)

// Интерфейс

type Repository interface {
	GetShortener(req *GetShortenerRequest) (*GetShortenerResponse, error)
	SetShortener(ctx context.Context, req *SetShortenerRequest) (*SetShortenerResponse, error)
	SetShortenerBatch(ctx context.Context, req *SetShortenerRequestBatch) (*SetShortenerResponseBatch, error)
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

type SetShortenerResponse struct {
	Code string
	URL  string
}

var (
	ErrSetShortenerAlreadyExists = errors.New("url already exists")
)

type SetShortenerRequestBatch struct {
	Rows []SetShortenerRequest
}

type SetShortenerResponseBatch struct {
	Rows []SetShortenerResponse
}

func NewStore(cfg config.Config) (Repository, error) {
	switch cfg.StoreType {
	case config.StoreTypeFile:
		if cfg.Filename != "" {
			return NewStoreFile(cfg)
		}
	case config.StoreTypeDB:
		if cfg.DBDsn != "" {
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

func (s *StoreVar) SetShortener(_ context.Context, req *SetShortenerRequest) (*SetShortenerResponse, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	// Проверка: уже существует
	for oldCode, oldURL := range s.shortener {
		if oldURL == req.URL {
			return &SetShortenerResponse{
				Code: oldCode,
				URL:  req.URL,
			}, ErrSetShortenerAlreadyExists
		}
	}

	// Запись в хранилище
	s.shortener[req.Code] = req.URL

	// Ответ
	return &SetShortenerResponse{
		Code: req.Code,
		URL:  req.URL,
	}, nil
}

func (s *StoreVar) SetShortenerBatch(_ context.Context, req *SetShortenerRequestBatch) (*SetShortenerResponseBatch, error) {
	var resp SetShortenerResponseBatch
	for _, req := range req.Rows {
		resprow, err := s.SetShortener(context.Background(), &req)
		if err != nil {
			return &SetShortenerResponseBatch{Rows: []SetShortenerResponse{0: *resprow}}, err
		}
		resp.Rows = append(resp.Rows, *resprow)
	}
	return &resp, nil
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

func (s *StoreFile) SetShortener(_ context.Context, req *SetShortenerRequest) (*SetShortenerResponse, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	// Проверка: уже существует
	for oldCode, oldURL := range s.shortener {
		if oldURL == req.URL {
			return &SetShortenerResponse{
				Code: oldCode,
				URL:  req.URL,
			}, ErrSetShortenerAlreadyExists
		}
	}

	// Запись в хранилище
	s.shortener[req.Code] = req.URL

	resp := &SetShortenerResponse{
		Code: req.Code,
		URL:  req.URL,
	}

	fileJSON := FileJSON{Code: req.Code, URL: req.URL}
	data, err := json.Marshal(&fileJSON)
	if err != nil {
		return resp, err
	}

	// записываем в буфер
	if _, err := s.writer.Write(data); err != nil {
		return resp, err
	}

	// добавляем перенос строки
	if err := s.writer.WriteByte('\n'); err != nil {
		return resp, err
	}

	// записываем буфер в файл
	s.writer.Flush()

	return resp, nil

}

func (s *StoreFile) SetShortenerBatch(_ context.Context, req *SetShortenerRequestBatch) (*SetShortenerResponseBatch, error) {
	var resp SetShortenerResponseBatch
	for _, req := range req.Rows {
		resprow, err := s.SetShortener(context.Background(), &req)
		if err != nil {
			return &SetShortenerResponseBatch{Rows: []SetShortenerResponse{0: *resprow}}, err
		}
		resp.Rows = append(resp.Rows, *resprow)
	}
	return &resp, nil
}

func (s *StoreFile) Ping() error {
	return nil
}

// Реализация с хранением в базе данных

type StoreDB struct {
	database *sql.DB
}

func NewStoreDB(cfg config.Config) (*StoreDB, error) {
	db, err := sql.Open("pgx", cfg.DBDsn)
	if err != nil {
		return nil, err
	}

	// Создаем таблицу
	_, err = db.Exec(
		"CREATE TABLE IF NOT EXISTS shortener (" +
			" code VARCHAR (10) PRIMARY KEY," +
			" url VARCHAR (255) NOT NULL" +
			" );")
	if err != nil {
		return nil, err
	}

	return &StoreDB{
		database: db,
	}, nil
}

func (s *StoreDB) GetShortener(req *GetShortenerRequest) (*GetShortenerResponse, error) {
	var url string
	row := s.database.QueryRow(
		"SELECT url FROM shortener"+
			" WHERE code = $1",
		req.Code)
	err := row.Scan(&url)
	if err != nil {
		return nil, err
	}

	return &GetShortenerResponse{
		URL: url,
	}, nil
}

func (s *StoreDB) SetShortener(ctx context.Context, req *SetShortenerRequest) (*SetShortenerResponse, error) {
	// Проверка: уже существует
	var oldCode string
	row := s.database.QueryRowContext(ctx,
		"SELECT code FROM shortener"+
			" WHERE url = $1",
		req.URL)
	err := row.Scan(&oldCode)
	if err == nil { // как ловить именно пустой результат, а не все ошибки БД? Ошибка нетипизирована error(*errors.errorString) *{s: "sql: no rows in result set"}
		return &SetShortenerResponse{
			Code: oldCode,
			URL:  req.URL,
		}, ErrSetShortenerAlreadyExists
	}

	query := "INSERT INTO shortener (code, url)" +
		" VALUES ($1, $2)" +
		" ON CONFLICT (code) DO NOTHING"
	_, err = s.database.ExecContext(ctx, query, // не понял как вернуть отсюда конфликтующую строку. Returning при конфликте возвращает пустоту
		req.Code, req.URL)
	if err != nil {
		return nil, err
	}

	return &SetShortenerResponse{
		Code: req.Code,
		URL:  req.URL,
	}, nil
}

func (s *StoreDB) SetShortenerBatch(ctx context.Context, req *SetShortenerRequestBatch) (*SetShortenerResponseBatch, error) {

	tx, err := s.database.Begin()
	if err != nil {
		return nil, err
	}

	var resp SetShortenerResponseBatch
	for _, req := range req.Rows {
		// Проверка: уже существует
		var oldCode string
		row := s.database.QueryRowContext(ctx,
			"SELECT code FROM shortener"+
				" WHERE url = $1",
			req.URL)
		err := row.Scan(&oldCode)
		if err == nil {
			return &SetShortenerResponseBatch{Rows: []SetShortenerResponse{0: {Code: oldCode, URL: req.URL}}}, ErrSetShortenerAlreadyExists
		}

		// Запись отдельной позиции
		_, err = tx.ExecContext(ctx,
			"INSERT INTO shortener AS t (code, url)"+
				" VALUES ($1, $2)"+
				" ON CONFLICT (code) DO NOTHING",
			req.Code, req.URL)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		resp.Rows = append(resp.Rows, SetShortenerResponse(req))
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *StoreDB) Ping() error {
	return s.database.Ping()
}
