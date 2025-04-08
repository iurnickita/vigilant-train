// Пакет service. Сервис
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/iurnickita/vigilant-train/internal/common/rand"
	"github.com/iurnickita/vigilant-train/internal/shortener/model"
	"github.com/iurnickita/vigilant-train/internal/shortener/repository"
)

// Service - интерфейс сервиса
type Service interface {
	GetShortener(code string) (model.Shortener, error)                // GetShortener читает короткую ссылку
	SetShortener(s model.Shortener) (model.Shortener, error)          // SetShortener создает короткую ссылку
	SetShortenerBatch(s []model.Shortener) ([]model.Shortener, error) // SetShortenerBatch создает короткую ссылку для набора данных
	Ping() error                                                      // Ping
	GetShortnerBatchUser(userCode string) ([]model.Shortener, error)  // GetShortenerBatch возвращает все ссылки, добавленные пользователем
	DeleteShortenerBatch(s []model.Shortener) error                   // DeleteShortenerBatch удаляет короткую ссылку
}

// Shortener - Сервис сокращения URL
type Shortener struct {
	store    repository.Repository
	toDelete chan []model.Shortener
}

func NewShortener(store repository.Repository) *Shortener {
	toDelete := make(chan []model.Shortener, 100)

	shortener := Shortener{
		store:    store,
		toDelete: toDelete,
	}

	go shortener.flushDeletes()

	return &shortener
}

// Ошибки пакета
var (
	ErrGetShortenerInvalidRequest = errors.New("invalid get Shortener request")
	ErrRepoFailed                 = errors.New("repo failed")
	ErrChanToDeleteIsFull         = errors.New("queue to delete is full")
)

func (service *Shortener) GetShortener(code string) (model.Shortener, error) {

	repositoryResp, err := service.store.GetShortener(code)
	if err != nil {
		if !errors.Is(err, repository.ErrGetShortenerNotFound) {
			return model.Shortener{}, fmt.Errorf("failed to fetch the Shortener result from the store: %w", err)
		}
	}

	return repositoryResp, nil
}

func (service *Shortener) SetShortener(s model.Shortener) (model.Shortener, error) {
	ctx := context.Background()

	s.Key.Code = rand.String(6)

	storeResp, err := service.store.SetShortener(ctx, s)
	if err != nil {
		return storeResp, err
	}

	return storeResp, nil
}

func (service *Shortener) SetShortenerBatch(s []model.Shortener) ([]model.Shortener, error) {
	ctx := context.Background()

	for i := range s {
		s[i].Key.Code = rand.String(6)
	}

	storeResp, err := service.store.SetShortenerBatch(ctx, s)

	return storeResp, err
}

func (service *Shortener) Ping() error {
	return service.store.Ping()
}

func (service *Shortener) GetShortnerBatchUser(userCode string) ([]model.Shortener, error) {
	ctx := context.Background()

	if userCode == "" {
		return nil, errors.New("userCode is empty")
	}

	return service.store.GetShortenerBatch(ctx, userCode)
}

func (service *Shortener) DeleteShortenerBatch(s []model.Shortener) error {
	service.toDelete <- s
	return nil
}

func (service *Shortener) flushDeletes() {
	ctx := context.Background()
	ticker := time.NewTicker(10 * time.Second)

	var toDelete []model.Shortener

	for {
		select {
		case s := <-service.toDelete:
			toDelete = append(toDelete, s...)
		case <-ticker.C:
			if len(toDelete) == 0 {
				continue
			}
			service.store.DeleteShortenerBatch(ctx, toDelete)
			toDelete = nil
		}
	}
}
