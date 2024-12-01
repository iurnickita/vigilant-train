package config

import (
	"flag"
	"os"
	"strings"

	handlersConfig "github.com/iurnickita/vigilant-train/internal/shortener/handlers/config"
)

type Config struct {
	Handlers handlersConfig.Config
}

func GetConfig() Config {
	cfg := Config{}

	flag.StringVar(&cfg.Handlers.ServerAddr, "a", "localhost:8080", "address of HTTP server")
	flag.StringVar(&cfg.Handlers.BaseAddr, "b", "localhost:8080", "address of short URL")
	flag.Parse()

	if envsrv := os.Getenv("SERVER_ADDRESS"); envsrv != "" {
		cfg.Handlers.ServerAddr = envsrv
	}
	if envbase := os.Getenv("BASE_URL"); envbase != "" {
		cfg.Handlers.BaseAddr = envbase
	}

	cfg.Handlers.ServerAddr = strings.TrimPrefix(cfg.Handlers.ServerAddr, "http://")
	cfg.Handlers.ServerAddr = strings.TrimPrefix(cfg.Handlers.ServerAddr, "http//")
	cfg.Handlers.BaseAddr = strings.TrimPrefix(cfg.Handlers.BaseAddr, "http://")
	cfg.Handlers.BaseAddr = strings.TrimPrefix(cfg.Handlers.BaseAddr, "http//")

	return cfg
}
