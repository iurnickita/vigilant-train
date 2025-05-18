// Пакет grpcserver. Обработчики grpc
package grpcserver

import (
	"context"
	"errors"
	"net"
	"sync"

	"github.com/iurnickita/vigilant-train/internal/shortener/auth"
	pb "github.com/iurnickita/vigilant-train/internal/shortener/grpc_server/proto"
	"github.com/iurnickita/vigilant-train/internal/shortener/grpc_server/server/config"
	"github.com/iurnickita/vigilant-train/internal/shortener/model"
	"github.com/iurnickita/vigilant-train/internal/shortener/repository"
	"github.com/iurnickita/vigilant-train/internal/shortener/service"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server grpc-обработчик
type Server struct {
	// нужно встраивать тип pb.Unimplemented<TypeName>
	// для совместимости с будущими версиями
	pb.UnimplementedShortenerServer

	config    config.Config
	shortener service.Service
	zaplog    *zap.Logger
	wg        sync.WaitGroup
}

// NewServer создает новый grpc сервер
func NewServer(config config.Config, shortener service.Service, zaplog *zap.Logger) *Server {
	return &Server{
		config:    config,
		shortener: shortener,
		zaplog:    zaplog,
	}
}

// Register возвращает новый токен
func (s *Server) Register(ctx context.Context, in *pb.Empty) (*pb.RegisterResponse, error) {
	token, err := auth.Register()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.RegisterResponse{Token: token}, nil
}

// GetShortener перенаправляет по короткой ссылке
func (s *Server) GetShortener(ctx context.Context, in *pb.GetShortenerRequest) (*pb.GetShortenerResponse, error) {

	resp, err := s.shortener.GetShortener(in.Code)
	if err != nil {
		if errors.Is(err, repository.ErrGetShortenerGone) {
			return nil, status.Errorf(codes.NotFound, err.Error())
		} else {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
	}

	var response pb.GetShortenerResponse
	response.Url = resp.Data.URL
	return &response, nil
}

// SetShortener создает короткую ссылку
func (s *Server) SetShortener(ctx context.Context, in *pb.SetShortenerRequest) (*pb.SetShortenerResponse, error) {
	// Код пользователя
	userCode := ctx.Value(auth.UserCodeKey).(auth.Key)

	// Получение полной URL
	resp, err := s.shortener.SetShortener(model.Shortener{
		Data: model.ShortenerData{URL: in.Url, User: string(userCode)},
	})
	if err != nil {
		if resp.Key.Code != "" {
			return &pb.SetShortenerResponse{Code: resp.Key.Code}, status.Error(codes.AlreadyExists, err.Error())
		} else {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	return &pb.SetShortenerResponse{Code: resp.Key.Code}, nil
}

// Обработчик DeleteShortenerBatch удаление набора ссылок
func (s *Server) DeleteShortenerBatch(ctx context.Context, in *pb.DeleteShortenerBatchRequest) (*pb.DeleteShortenerBatchResponse, error) {
	// получение id пользователя
	userCode := ctx.Value(auth.UserCodeKey).(auth.Key)

	// Конвертация
	arr := make([]model.Shortener, 0, len(in.Code))
	for _, code := range in.Code {
		arr = append(arr, model.Shortener{Key: model.ShortenerKey{Code: code}, Data: model.ShortenerData{User: string(userCode)}})
	}

	// Вызов метода сервиса
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.shortener.DeleteShortenerBatch(arr)
	}()

	return &pb.DeleteShortenerBatchResponse{}, nil
}

// Serve - запуск сервера
func Serve(cfg config.Config, shortener service.Service, zaplog *zap.Logger) error {
	// определяем порт для сервера
	listen, err := net.Listen("tcp", ":3200")
	if err != nil {
		return err
	}
	// создаём gRPC-сервер
	s := grpc.NewServer(grpc.UnaryInterceptor(auth.AuthUnaryInterceptor))
	// создание обработчика
	h := NewServer(cfg, shortener, zaplog)
	// регистрируем сервис
	pb.RegisterShortenerServer(s, h)

	zaplog.Info("Сервер gRPC начал работу")
	// получаем запрос gRPC
	if err := s.Serve(listen); err != nil {
		return err
	}
	return nil
}

// Установка grpc
// https://grpc.io/docs/languages/go/quickstart/

// Генерация go-файлов для grpc сервиса
// cd internal/grpc_server
// protoc --go_out=. --go_opt=paths=source_relative \
// --go-grpc_out=. --go-grpc_opt=paths=source_relative \
// proto/server.proto
