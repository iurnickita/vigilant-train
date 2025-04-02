package main

import (
	"log"

	"github.com/iurnickita/vigilant-train/internal/shortener/config"
	"github.com/iurnickita/vigilant-train/internal/shortener/handlers"
	"github.com/iurnickita/vigilant-train/internal/shortener/logger"
	"github.com/iurnickita/vigilant-train/internal/shortener/repository"
	"github.com/iurnickita/vigilant-train/internal/shortener/service"
	//_ "net/http/pprof"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg := config.GetConfig()

	zaplog, err := logger.NewZapLog(cfg.Logger)
	if err != nil {
		return err
	}

	store, err := repository.NewStore(cfg.Repository)
	if err != nil {
		return err
	}

	shortenerService := service.NewShortener(store)

	// pprof run
	//go http.ListenAndServe(cfg.Pprof.ServerAddr, nil)

	return handlers.Serve(cfg.Handlers, shortenerService, zaplog)
	// ловить ошибку, Defer db.close
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
