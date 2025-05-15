// Пакет service. Сервис
package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/iurnickita/vigilant-train/internal/common/rand"
	"github.com/iurnickita/vigilant-train/internal/shortener/model"
	"github.com/iurnickita/vigilant-train/internal/shortener/repository"
)

// Service - интерфейс сервиса
type Service interface {
	// GetShortener читает короткую ссылку
	GetShortener(code string) (model.Shortener, error)
	// SetShortener создает короткую ссылку
	SetShortener(s model.Shortener) (model.Shortener, error)
	// SetShortenerBatch создает короткую ссылку для набора данных
	SetShortenerBatch(s []model.Shortener) ([]model.Shortener, error)
	// Ping
	Ping() error
	// GetShortenerBatch возвращает все ссылки, добавленные пользователем
	GetShortnerBatchUser(userCode string) ([]model.Shortener, error)
	// DeleteShortenerBatch удаляет короткую ссылку
	DeleteShortenerBatch(s []model.Shortener) error
	// GetStats возвращает статистические данные
	GetStats(ctx context.Context) (model.Stats, error)
	// Shutdown завершает и ожидает все процессы
	Shutdown()
}

// Shortener - Сервис сокращения URL
type Shortener struct {
	store    repository.Repository
	toDelete chan []model.Shortener
	shutdown chan bool
	wg       sync.WaitGroup
}

// NewShortener конструктор
func NewShortener(store repository.Repository) *Shortener {
	toDelete := make(chan []model.Shortener, 100)
	shutdown := make(chan bool)

	shortener := Shortener{
		store:    store,
		toDelete: toDelete,
		shutdown: shutdown,
	}

	shortener.wg.Add(1)
	go shortener.flushDeletes()

	return &shortener
}

// Ошибки пакета
var (
	ErrGetShortenerInvalidRequest = errors.New("invalid get Shortener request")
	ErrRepoFailed                 = errors.New("repo failed")
	ErrChanToDeleteIsFull         = errors.New("queue to delete is full")
)

// GetShortener читает короткую ссылку
func (service *Shortener) GetShortener(code string) (model.Shortener, error) {

	repositoryResp, err := service.store.GetShortener(code)
	if err != nil {
		if !errors.Is(err, repository.ErrGetShortenerNotFound) {
			return model.Shortener{}, fmt.Errorf("failed to fetch the Shortener result from the store: %w", err)
		}
	}

	return repositoryResp, nil
}

// SetShortener создает короткую ссылку
func (service *Shortener) SetShortener(s model.Shortener) (model.Shortener, error) {
	ctx := context.Background()

	s.Key.Code = rand.String(6)

	storeResp, err := service.store.SetShortener(ctx, s)
	if err != nil {
		return storeResp, err
	}

	return storeResp, nil
}

// SetShortenerBatch создает короткую ссылку для набора данных
func (service *Shortener) SetShortenerBatch(s []model.Shortener) ([]model.Shortener, error) {
	ctx := context.Background()

	for i := range s {
		s[i].Key.Code = rand.String(6)
	}

	storeResp, err := service.store.SetShortenerBatch(ctx, s)

	return storeResp, err
}

// Ping
func (service *Shortener) Ping() error {
	return service.store.Ping()
}

// GetShortenerBatch возвращает все ссылки, добавленные пользователем
func (service *Shortener) GetShortnerBatchUser(userCode string) ([]model.Shortener, error) {
	ctx := context.Background()

	if userCode == "" {
		return nil, errors.New("userCode is empty")
	}

	return service.store.GetShortenerBatch(ctx, userCode)
}

// DeleteShortenerBatch удаляет короткую ссылку
func (service *Shortener) DeleteShortenerBatch(s []model.Shortener) error {
	service.toDelete <- s
	return nil
}

func (service *Shortener) flushDeletes() {
	defer service.wg.Done()
	ctx := context.Background()
	ticker := time.NewTicker(3 * time.Second)

	var toDelete []model.Shortener
	var shutdown bool

L:
	for {
		select {
		// Сигнал завершения
		case <-service.shutdown:
			shutdown = true
		// Накопление очереди к удалению
		case s := <-service.toDelete:
			toDelete = append(toDelete, s...)
		// Удаление набора из очереди
		case <-ticker.C:
			if len(toDelete) == 0 {
				continue
			}
			service.store.DeleteShortenerBatch(ctx, toDelete)

			toDelete = nil

			// Завершение работы
			if shutdown {
				break L
			}
		}
	}
}

// GetStats возвращает статистические данные
func (service *Shortener) GetStats(ctx context.Context) (model.Stats, error) {
	return service.store.GetStats(ctx)
}

// Shutdown завершает и ожидает все процессы
func (service *Shortener) Shutdown() {
	// Передача сигнала завершения
	service.shutdown <- true
	// Ожидание завершения
	service.wg.Wait()
	// Закрытие хранилища
	service.store.Close()
}
