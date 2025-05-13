// Пакет repository. Хранилище данных
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

// Repository - интерфейс хранилища
type Repository interface {
	// GetShortener читает короткую ссылку
	GetShortener(code string) (model.Shortener, error)
	// SetShortener создает короткую ссылку
	SetShortener(ctx context.Context, s model.Shortener) (model.Shortener, error)
	// SetShortenerBatch создает короткую ссылку для набора данных
	SetShortenerBatch(ctx context.Context, s []model.Shortener) ([]model.Shortener, error)
	// Ping
	Ping() error
	// GetShortenerBatch возвращает все ссылки, добавленные пользователем
	GetShortenerBatch(ctx context.Context, userCode string) ([]model.Shortener, error)
	// DeleteShortenerBatch удаляет короткую ссылку
	DeleteShortenerBatch(ctx context.Context, s []model.Shortener) error
	// GetStats возвращает статистические данные
	GetStats(ctx context.Context) (model.Stats, error)
}

// NewStore возвращает одну из сущестующих реализаций хранилища в зависимости от конфигурации сервиса
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

// Ошибки пакета
var (
	ErrGetShortenerNotFound      = errors.New("data not found")
	ErrSetShortenerAlreadyExists = errors.New("url already exists")
	ErrGetShortenerGone          = errors.New("code is deleted")
)

// newErrGetShortenerNotFound - подробная ошибка NotFound
func newErrGetShortenerNotFound(code string) error {
	return fmt.Errorf("%w for code = %s", ErrGetShortenerNotFound, code)
}

// StoreVar - Реализация с хранением в переменной
type StoreVar struct {
	mux       *sync.Mutex
	shortener map[model.ShortenerKey]model.ShortenerData
}

// NewStoreVar - конструктор хранилища
func NewStoreVar(cfg config.Config) (*StoreVar, error) {
	return &StoreVar{
		mux:       &sync.Mutex{},
		shortener: make(map[model.ShortenerKey]model.ShortenerData),
	}, nil
}

// GetShortener читает короткую ссылку
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

// SetShortener создает короткую ссылку
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

// SetShortenerBatch создает короткую ссылку для набора данных
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

// Ping
func (store *StoreVar) Ping() error {
	return nil
}

// GetShortenerBatch возвращает все ссылки, добавленные пользователем
func (store *StoreVar) GetShortenerBatch(_ context.Context, userCode string) ([]model.Shortener, error) {
	var resp []model.Shortener
	for key, data := range store.shortener {
		if data.User == userCode || userCode == "" {
			resp = append(resp, model.Shortener{Key: key, Data: data})
		}
	}
	return resp, nil
}

// DeleteShortenerBatch удаляет короткую ссылку
func (store *StoreVar) DeleteShortenerBatch(_ context.Context, s []model.Shortener) error {
	for _, s := range s {
		store.shortener[s.Key] = model.ShortenerData{}
	}
	return nil
}

// GetStats возвращает статистические данные
func (store *StoreVar) GetStats(ctx context.Context) (model.Stats, error) {
	var stats model.Stats

	// кол-во сокращенных ссылок
	stats.URLs = len(store.shortener)

	// кол-во пользователей
	// *здесь не помешал бы бинарный поиск
	users := make([]string, 0, 10)
	for _, sh := range store.shortener {
		exists := false
		for _, us := range users {
			if us == sh.User {
				exists = true
				break
			}
		}
		if !exists {
			users = append(users, sh.User)
		}
	}
	stats.Users = len(users)

	return stats, nil
}

// StoreFile - Реализация с хранением в файле
type StoreFile struct {
	mux       *sync.Mutex
	shortener map[model.ShortenerKey]model.ShortenerData
	writer    *bufio.Writer
}

// FileJSON Структура JSON-файла для хранения
type FileJSON struct {
	Code string `json:"code"`
	URL  string `json:"url"`
	User string `json:"user"`
}

// NewStoreFile - конструктор хранилища
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

// GetShortener читает короткую ссылку
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

// SetShortener создает короткую ссылку
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

// SetShortenerBatch создает короткую ссылку для набора данных
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

// Ping
func (store *StoreFile) Ping() error {
	return nil
}

// GetShortenerBatch возвращает все ссылки, добавленные пользователем
func (store *StoreFile) GetShortenerBatch(_ context.Context, userCode string) ([]model.Shortener, error) {
	var resp []model.Shortener
	for key, data := range store.shortener {
		if data.User == userCode || userCode == "" {
			resp = append(resp, model.Shortener{Key: key, Data: data})
		}
	}
	return resp, nil
}

// DeleteShortenerBatch удаляет короткую ссылку
func (store *StoreFile) DeleteShortenerBatch(_ context.Context, s []model.Shortener) error {
	return nil
}

// GetStats возвращает статистические данные
func (store *StoreFile) GetStats(ctx context.Context) (model.Stats, error) {
	var stats model.Stats

	// кол-во сокращенных ссылок
	stats.URLs = len(store.shortener)

	// кол-во пользователей
	// *здесь не помешал бы бинарный поиск
	users := make([]string, 0, 10)
	for _, sh := range store.shortener {
		exists := false
		for _, us := range users {
			if us == sh.User {
				exists = true
				break
			}
		}
		if !exists {
			users = append(users, sh.User)
		}
	}
	stats.Users = len(users)

	return stats, nil
}

// StoreDB - Реализация с хранением в базе данных
type StoreDB struct {
	database *sql.DB
}

// NewStoreDB - конструктор хранилища
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

// GetShortener читает короткую ссылку
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

// SetShortener создает короткую ссылку
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

// SetShortenerBatch создает короткую ссылку для набора данных
func (store *StoreDB) SetShortenerBatch(ctx context.Context, s []model.Shortener) ([]model.Shortener, error) {

	tx, err := store.database.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var respSBatch []model.Shortener
	for _, reqS := range s {
		// Проверка: уже существует
		var oldCode string
		row := store.database.QueryRowContext(ctx,
			"SELECT code FROM shortener"+
				" WHERE url = $1"+
				" FOR UPDATE",
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

// Ping
func (store *StoreDB) Ping() error {
	return store.database.Ping()
}

// GetShortenerBatch возвращает все ссылки, добавленные пользователем
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
	defer rows.Close()
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

// DeleteShortenerBatch удаляет короткую ссылку
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

// GetStats возвращает статистические данные
func (store *StoreDB) GetStats(ctx context.Context) (model.Stats, error) {
	var stats model.Stats

	row := store.database.QueryRowContext(ctx,
		"SELECT count(code)"+
			"FROM shortener"+
			"WHERE del_flag = FALSE"+
			"LIMIT 1")
	if err := row.Scan(stats.URLs); err != nil {
		return model.Stats{}, err
	}

	row = store.database.QueryRowContext(ctx,
		"SELECT count(sh.uuid)"+
			"FROM (SELECT DISTINCT uuid FROM shortener) sh")
	if err := row.Scan(stats.Users); err != nil {
		return model.Stats{}, err
	}

	return stats, nil
}
