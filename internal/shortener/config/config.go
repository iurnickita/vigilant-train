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
}

func GetConfig() Config {
	cfg := Config{}

	flag.StringVar(&cfg.Handlers.ServerAddr, "a", "localhost:8080", "address of HTTP server")
	flag.StringVar(&cfg.Handlers.BaseAddr, "b", "localhost:8080", "address of short URL")
	flag.StringVar(&cfg.Logger.LogLevel, "l", "info", "log level")
	flag.StringVar(&cfg.Repository.StoreType, "s", repositoryConfig.StoreTypeFile, "storage type: 0 Var, 1 File (default)")
	flag.StringVar(&cfg.Repository.Filename, "f", repositoryConfig.DefaultFilename, "filename (if s = 1)")
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
	if envstore := os.Getenv("STORAGE_TYPE"); envstore != "" {
		cfg.Repository.StoreType = envstore
	}
	if envstore := os.Getenv("FILE_STORAGE_PATH"); envstore != "" {
		cfg.Repository.StoreType = envstore
	}

	// костыль для кривых данных
	cfg.Handlers.ServerAddr = strings.TrimPrefix(cfg.Handlers.ServerAddr, "http://")
	cfg.Handlers.ServerAddr = strings.TrimPrefix(cfg.Handlers.ServerAddr, "http//")
	cfg.Handlers.BaseAddr = strings.TrimPrefix(cfg.Handlers.BaseAddr, "http://")
	cfg.Handlers.BaseAddr = strings.TrimPrefix(cfg.Handlers.BaseAddr, "http//")

	return cfg
}
