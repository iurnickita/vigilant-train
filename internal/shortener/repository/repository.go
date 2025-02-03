package repository

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/iurnickita/vigilant-train/internal/shortener/model"
	"github.com/iurnickita/vigilant-train/internal/shortener/repository/config"
)

// Интерфейс

type Repository interface {
	GetShortener(code string) (model.Shortener, error)
	SetShortener(ctx context.Context, s model.Shortener) (model.Shortener, error)
	SetShortenerBatch(ctx context.Context, s []model.Shortener) ([]model.Shortener, error)
	Ping() error
	GetShortenerBatch(ctx context.Context, userCode string) ([]model.Shortener, error)
	DeleteShortenerBatch(ctx context.Context, s []model.Shortener) error
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

var (
	ErrGetShortenerNotFound      = errors.New("data not found")
	ErrSetShortenerAlreadyExists = errors.New("url already exists")
	ErrGetShortenerGone          = errors.New("code is deleted")
)

func newErrGetShortenerNotFound(code string) error {
	return fmt.Errorf("%w for code = %s", ErrGetShortenerNotFound, code)
}

// Реализация с хранением в переменной

type StoreVar struct {
	mux       *sync.Mutex
	shortener map[model.ShortenerKey]model.ShortenerData
}

func NewStoreVar(cfg config.Config) (*StoreVar, error) {
	return &StoreVar{
		mux:       &sync.Mutex{},
		shortener: make(map[model.ShortenerKey]model.ShortenerData),
	}, nil
}

func (store *StoreVar) GetShortener(code string) (model.Shortener, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	key := model.ShortenerKey{Code: code}
	data, ok := store.shortener[key]
	if !ok {
		return model.Shortener{}, newErrGetShortenerNotFound(code)
	}
	return model.Shortener{
		Key:  key,
		Data: data,
	}, nil
}

func (store *StoreVar) SetShortener(_ context.Context, s model.Shortener) (model.Shortener, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	// Проверка: уже существует
	for oldKey, oldData := range store.shortener {
		if oldData.URL == s.Data.URL {
			return model.Shortener{
				Key:  oldKey,
				Data: oldData,
			}, ErrSetShortenerAlreadyExists
		}
	}

	// Запись в хранилище
	store.shortener[s.Key] = s.Data

	// Ответ
	return s, nil
}

func (store *StoreVar) SetShortenerBatch(_ context.Context, s []model.Shortener) ([]model.Shortener, error) {
	var respSBatch []model.Shortener
	for _, reqS := range s {
		respS, err := store.SetShortener(context.Background(), reqS)
		if err != nil {
			return []model.Shortener{1: respS}, err
		}
		respSBatch = append(respSBatch, respS)
	}
	return respSBatch, nil
}

func (store *StoreVar) Ping() error {
	return nil
}

func (store *StoreVar) GetShortenerBatch(_ context.Context, userCode string) ([]model.Shortener, error) {
	var resp []model.Shortener
	for key, data := range store.shortener {
		if data.User == userCode || userCode == "" {
			resp = append(resp, model.Shortener{Key: key, Data: data})
		}
	}
	return resp, nil
}

func (store *StoreVar) DeleteShortenerBatch(_ context.Context, s []model.Shortener) error {
	for _, s := range s {
		store.shortener[s.Key] = model.ShortenerData{}
	}
	return nil
}

// Реализация с хранением в файле

type StoreFile struct {
	mux       *sync.Mutex
	shortener map[model.ShortenerKey]model.ShortenerData
	writer    *bufio.Writer
}

type FileJSON struct {
	Code string `json:"code"`
	URL  string `json:"url"`
	User string `json:"user"`
}

func NewStoreFile(cfg config.Config) (*StoreFile, error) {
	file, err := os.OpenFile(cfg.Filename, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	// Наполнение мапы из файла
	scanner := bufio.NewScanner(file)
	var fileJSON FileJSON
	shortener := map[model.ShortenerKey]model.ShortenerData{}
	for scanner.Scan() {
		if err := json.Unmarshal(scanner.Bytes(), &fileJSON); err == nil && fileJSON.Code != "" {
			shortener[model.ShortenerKey{Code: fileJSON.Code}] =
				model.ShortenerData{URL: fileJSON.URL, User: fileJSON.User}
		}
	}

	return &StoreFile{
		mux:       &sync.Mutex{},
		shortener: shortener,
		writer:    bufio.NewWriter(file),
	}, nil
}

func (store *StoreFile) GetShortener(code string) (model.Shortener, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	Key := model.ShortenerKey{Code: code}
	data, ok := store.shortener[Key]
	if !ok {
		return model.Shortener{}, newErrGetShortenerNotFound(code)
	}
	return model.Shortener{
		Key:  Key,
		Data: data,
	}, nil
}

func (store *StoreFile) SetShortener(_ context.Context, s model.Shortener) (model.Shortener, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	// Проверка: уже существует
	for oldKey, oldData := range store.shortener {
		if oldData.URL == s.Data.URL {
			return model.Shortener{
				Key:  oldKey,
				Data: oldData,
			}, ErrSetShortenerAlreadyExists
		}
	}

	// Запись в хранилище
	store.shortener[s.Key] = s.Data
	respS := s

	fileJSON := FileJSON{Code: s.Key.Code, URL: s.Data.URL, User: s.Data.User}
	data, err := json.Marshal(&fileJSON)
	if err != nil {
		return respS, err
	}

	// записываем в буфер
	if _, err := store.writer.Write(data); err != nil {
		return respS, err
	}

	// добавляем перенос строки
	if err := store.writer.WriteByte('\n'); err != nil {
		return respS, err
	}

	// записываем буфер в файл
	store.writer.Flush()

	return respS, nil

}

func (store *StoreFile) SetShortenerBatch(_ context.Context, s []model.Shortener) ([]model.Shortener, error) {
	var respSBatch []model.Shortener
	for _, reqS := range s {
		respS, err := store.SetShortener(context.Background(), reqS)
		if err != nil {
			return []model.Shortener{1: respS}, err
		}
		respSBatch = append(respSBatch, respS)
	}
	return respSBatch, nil
}

func (store *StoreFile) Ping() error {
	return nil
}

func (store *StoreFile) GetShortenerBatch(_ context.Context, userCode string) ([]model.Shortener, error) {
	var resp []model.Shortener
	for key, data := range store.shortener {
		if data.User == userCode || userCode == "" {
			resp = append(resp, model.Shortener{Key: key, Data: data})
		}
	}
	return resp, nil
}

func (store *StoreFile) DeleteShortenerBatch(_ context.Context, s []model.Shortener) error {
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
			" url VARCHAR (255) NOT NULL," +
			" uuid VARCHAR (10) DEFAULT NULL," +
			" del_flag BOOLEAN DEFAULT FALSE" +
			" );")
	if err != nil {
		return nil, err
	}

	return &StoreDB{
		database: db,
	}, nil
}

func (store *StoreDB) GetShortener(code string) (model.Shortener, error) {
	var url string
	var delFlag bool
	row := store.database.QueryRow(
		"SELECT url, del_flag FROM shortener"+
			" WHERE code = $1",
		code)
	err := row.Scan(&url, &delFlag)
	if err != nil {
		return model.Shortener{}, err
	}
	if delFlag {
		return model.Shortener{}, ErrGetShortenerGone
	}

	return model.Shortener{
		Key:  model.ShortenerKey{Code: code},
		Data: model.ShortenerData{URL: url},
	}, nil
}

func (store *StoreDB) SetShortener(ctx context.Context, s model.Shortener) (model.Shortener, error) {
	// Проверка: уже существует
	var oldCode string
	row := store.database.QueryRowContext(ctx,
		"SELECT code FROM shortener"+
			" WHERE url = $1",
		s.Data.URL)
	err := row.Scan(&oldCode)
	if err == nil { // как ловить именно пустой результат, а не все ошибки БД? Ошибка нетипизирована error(*errors.errorString) *{s: "sql: no rows in result set"}
		return model.Shortener{
			Key:  model.ShortenerKey{Code: oldCode},
			Data: model.ShortenerData{URL: s.Data.URL},
		}, ErrSetShortenerAlreadyExists
	}

	query := "INSERT INTO shortener (code, url, uuid)" +
		" VALUES ($1, $2, $3)" +
		" ON CONFLICT (code) DO NOTHING"
	_, err = store.database.ExecContext(ctx, query, // не понял как вернуть отсюда конфликтующую строку. Returning при конфликте возвращает пустоту
		s.Key.Code, s.Data.URL, s.Data.User)
	if err != nil {
		return model.Shortener{}, err
	}

	return s, nil
}

func (store *StoreDB) SetShortenerBatch(ctx context.Context, s []model.Shortener) ([]model.Shortener, error) {

	tx, err := store.database.Begin()
	if err != nil {
		return nil, err
	}

	var respSBatch []model.Shortener
	for _, reqS := range s {
		// Проверка: уже существует
		var oldCode string
		row := store.database.QueryRowContext(ctx,
			"SELECT code FROM shortener"+
				" WHERE url = $1",
			reqS.Data.URL)
		err := row.Scan(&oldCode)
		if err == nil {
			return []model.Shortener{1: {Key: model.ShortenerKey{Code: oldCode},
					Data: model.ShortenerData{URL: reqS.Data.URL}}},
				ErrSetShortenerAlreadyExists
		}

		// Запись отдельной позиции
		_, err = tx.ExecContext(ctx,
			"INSERT INTO shortener AS t (code, url, uuid)"+
				" VALUES ($1, $2, $3)"+
				" ON CONFLICT (code) DO NOTHING",
			reqS.Key.Code, reqS.Data.URL, reqS.Data.User)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		respSBatch = append(respSBatch, reqS)
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return respSBatch, nil
}

func (store *StoreDB) Ping() error {
	return store.database.Ping()
}

func (store *StoreDB) GetShortenerBatch(ctx context.Context, userCode string) ([]model.Shortener, error) {
	var resp []model.Shortener

	rows, err := store.database.QueryContext(ctx,
		"SELECT code, url FROM shortener"+
			" WHERE uuid = $1"+ // как сделать опциональное условие
			" AND del_flag = FALSE",
		userCode)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var respRow model.Shortener
		err := rows.Scan(&respRow.Key.Code, &respRow.Data.URL)
		if err != nil {
			return nil, err
		}
		resp = append(resp, respRow)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return resp, nil

}

func (store *StoreDB) DeleteShortenerBatch(ctx context.Context, s []model.Shortener) error {

	var values []string
	var args []any

	for i, s := range s {
		base := i * 2
		params := fmt.Sprintf("($%d, $%d)", base+1, base+2)
		values = append(values, params)
		args = append(args, s.Key.Code, s.Data.User)
	}

	query := "UPDATE shortener AS s" +
		" SET del_flag = TRUE" +
		" FROM (values " +
		strings.Join(values, ",") +
		" ) AS k(code, uuid) " +
		" WHERE s.code = k.code" +
		"   AND s.uuid = k.uuid"

	_, err := store.database.ExecContext(ctx, query, args...)

	return err
}
