package config

import (
	"flag"
	"os"
	"strings"

	handlersConfig "github.com/iurnickita/vigilant-train/internal/shortener/handlers/config"
	loggerConfig "github.com/iurnickita/vigilant-train/internal/shortener/logger/config"
	repositoryConfig "github.com/iurnickita/vigilant-train/internal/shortener/repository/config"
)

type Config struct {
	Handlers   handlersConfig.Config
	Logger     loggerConfig.Config
	Repository repositoryConfig.Config
	Pprof      PprofConfig
}

type PprofConfig struct {
	ServerAddr string
}

func GetConfig() Config {
	cfg := Config{}

	flag.StringVar(&cfg.Handlers.ServerAddr, "a", "localhost:8080", "address of HTTP server")
	flag.StringVar(&cfg.Handlers.BaseAddr, "b", "localhost:8080", "address of short URL")
	flag.StringVar(&cfg.Logger.LogLevel, "l", "info", "log level")
	flag.StringVar(&cfg.Repository.Filename, "f", "", "file path")
	flag.StringVar(&cfg.Repository.DBDsn, "d", "", "database dsn")
	flag.Parse()

	if envsrv := os.Getenv("SERVER_ADDRESS"); envsrv != "" {
		cfg.Handlers.ServerAddr = envsrv
	}
	if envbase := os.Getenv("BASE_URL"); envbase != "" {
		cfg.Handlers.BaseAddr = envbase
	}
	if envlevel := os.Getenv("LOG_LEVEL"); envlevel != "" {
		cfg.Logger.LogLevel = envlevel
	}
	if envspath := os.Getenv("FILE_STORAGE_PATH"); envspath != "" {
		cfg.Repository.Filename = envspath
	}
	if envdbase := os.Getenv("DATABASE_DSN"); envdbase != "" {
		cfg.Repository.DBDsn = envdbase
	}

	/* if cfg.Repository.DBDsn == "" {
		cfg.Repository.DBDsn = "host=localhost user=bob password=bob dbname=shortener sslmode=disable"
	} */

	if cfg.Repository.DBDsn != "" {
		cfg.Repository.StoreType = repositoryConfig.StoreTypeDB
	} else if cfg.Repository.Filename != "" {
		cfg.Repository.StoreType = repositoryConfig.StoreTypeFile
	} else {
		cfg.Repository.StoreType = repositoryConfig.StoreTypeVar
	}

	cfg.Pprof.ServerAddr = "localhost:6060"

	// костыль для кривых данных
	cfg.Handlers.ServerAddr = strings.TrimPrefix(cfg.Handlers.ServerAddr, "http://")
	cfg.Handlers.ServerAddr = strings.TrimPrefix(cfg.Handlers.ServerAddr, "http//")
	cfg.Handlers.BaseAddr = strings.TrimPrefix(cfg.Handlers.BaseAddr, "http://")
	cfg.Handlers.BaseAddr = strings.TrimPrefix(cfg.Handlers.BaseAddr, "http//")

	return cfg
}
