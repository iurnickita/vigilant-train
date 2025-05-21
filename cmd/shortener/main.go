// Shortener - Сервис сокращения URL
package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"sync"

	"github.com/iurnickita/vigilant-train/internal/shortener/config"
	grpc "github.com/iurnickita/vigilant-train/internal/shortener/grpc_server/server"
	"github.com/iurnickita/vigilant-train/internal/shortener/handlers"
	"github.com/iurnickita/vigilant-train/internal/shortener/logger"
	"github.com/iurnickita/vigilant-train/internal/shortener/repository"
	"github.com/iurnickita/vigilant-train/internal/shortener/service"
)

var (
	buildVersion string
	buildDate    string
	bulidCommit  string
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// Run service
func run() error {
	// Флаги сборки (флаги линковщика)
	fmt.Printf("buildVersion: %s\n", fillEmpty(buildVersion))
	fmt.Printf("buildDate: %s\n", fillEmpty(buildDate))
	fmt.Printf("bulidCommit: %s\n", fillEmpty(bulidCommit))

	// Config
	cfg := config.GetConfig()

	// Log
	zaplog, err := logger.NewZapLog(cfg.Logger)
	if err != nil {
		return err
	}

	// Store
	store, err := repository.NewStore(cfg.Repository)
	if err != nil {
		return err
	}

	// Service
	shortenerService := service.NewShortener(store)

	// pprof run
	if cfg.Pprof.ServerAddr != "" {
		go http.ListenAndServe(cfg.Pprof.ServerAddr, nil)
	}

	// Handlers
	var wg sync.WaitGroup
	wg.Add(2)
	// HTTP server run
	go func() {
		defer wg.Done()
		err := handlers.Serve(cfg.Handlers, shortenerService, zaplog)
		if err != nil {
			zaplog.Error(err.Error())
		}
	}()
	// gRPC server run
	go func() {
		defer wg.Done()
		err := grpc.Serve(cfg.GRPCServer, shortenerService, zaplog)
		if err != nil {
			zaplog.Error(err.Error())
		}
	}()

	wg.Wait()
	return nil
}

func fillEmpty(s string) string {
	if s == "" {
		s = "N/A"
	}
	return s
}

// curl -v -X POST -d https://practicum.yandex.ru/ http://localhost:8080/
// curl -v --json '{"url": "https://practicum.yandex.ru"}' http://localhost:8080/api/shorten
// curl -v --json '[{"correlation_id": "15", "original_url": "https://www.postgresql.org/docs/current/sql-load.html"}]' http://localhost:8080/api/shorten/batch
// curl -v http://localhost:8080/ping

// curl -v --cookie "shortenerUserToken=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJVc2VyQ29kZSI6Ik9wbjQifQ.4l6jDxxtNqK25Lr8ilCdvEdkT-fTSvj90FJwCnSb5q4" http://localhost:8080/api/user/urls
// curl -v -X DELETE --json '["mLIECn"]' --cookie "shortenerUserToken=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJVc2VyQ29kZSI6InpiY20ifQ.-Sq1MQxXGibVii7B3TdlF-LGM7TaL7Bttf6IQbqsrnw" http://localhost:8080/api/user/urls

// Профилирование. Инструмент pprof
//
// go tool pprof -http=":9090" -seconds=30 http://localhost:6060/debug/pprof/profile
// go tool pprof -http=":9090" -seconds=30 http://localhost:6060/debug/pprof/heap
// Запись результатов мониторинга в файл
// curl -o profiles/base.pprof http://localhost:6060/debug/pprof/heap?seconds=30
// Просмотр из файла
// go tool pprof -http=":9090" profiles/base.pprof
// Сравнение результатов
// go tool pprof -top -diff_base=profiles/base.pprof profiles/result.pprof

// Флаги сборки (флаги линковщика)
// go run -ldflags "-X main.buildVersion=v1.0.1" main.go
