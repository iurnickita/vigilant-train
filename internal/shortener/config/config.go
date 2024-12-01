package config

import (
	"flag"

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
	return cfg
}
