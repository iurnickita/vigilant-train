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

type Service interface {
	GetShortener(code string) (model.Shortener, error)
	SetShortener(s model.Shortener) (model.Shortener, error)
	SetShortenerBatch(s []model.Shortener) ([]model.Shortener, error)
	Ping() error
	GetNewUserCode() string
	GetShortnerBatchUser(userCode string) ([]model.Shortener, error)
	DeleteShortenerBatch(s []model.Shortener) error
}

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

func (service *Shortener) GetNewUserCode() string {
	return rand.String(4)
}

func (service *Shortener) GetShortnerBatchUser(userCode string) ([]model.Shortener, error) {
	ctx := context.Background()

	if userCode == "" {
		return nil, errors.New("userCode is empty")
	}

	return service.store.GetShortenerBatch(ctx, userCode)
}

func (service *Shortener) DeleteShortenerBatch(s []model.Shortener) error {
	if len(service.toDelete) >= cap(service.toDelete) {
		return ErrChanToDeleteIsFull
	}
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
