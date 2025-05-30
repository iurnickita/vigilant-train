// Пакет config. Конфигурация с помощью флагов/переменных среды/значений по умолчанию
package config

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"strings"

	grpcServerConfig "github.com/iurnickita/vigilant-train/internal/shortener/grpc_server/server/config"
	handlersConfig "github.com/iurnickita/vigilant-train/internal/shortener/handlers/config"
	loggerConfig "github.com/iurnickita/vigilant-train/internal/shortener/logger/config"
	repositoryConfig "github.com/iurnickita/vigilant-train/internal/shortener/repository/config"
)

// Config - общая конфигурация
type Config struct {
	Handlers   handlersConfig.Config
	GRPCServer grpcServerConfig.Config
	Logger     loggerConfig.Config
	Repository repositoryConfig.Config
	Pprof      PprofConfig
}

// PprofConfig - конфигурация профилировщика
type PprofConfig struct {
	ServerAddr string
}

// GetConfig собирает конфигурацию сервиса
func GetConfig() Config {
	cfg := Config{}
	var cfgFileName string

	// Флаги
	flag.StringVar(&cfg.Handlers.ServerAddr, "a", "localhost:8080", "address of HTTP server")
	flag.StringVar(&cfg.Handlers.BaseAddr, "b", "localhost:8080", "address of short URL")
	flag.StringVar(&cfg.Logger.LogLevel, "l", "info", "log level")
	flag.StringVar(&cfg.Repository.Filename, "f", "", "file path")
	flag.StringVar(&cfg.Repository.DBDsn, "d", "", "database dsn")
	flag.StringVar(&cfg.Pprof.ServerAddr, "p", "", "address of Pprof server") // "localhost:6060" - не заполняю по умолчанию, потому что занятый порт мешает тестам
	flag.BoolVar(&cfg.Handlers.EnableHTTPS, "s", false, "enable HTTPS on server")
	flag.StringVar(&cfg.Handlers.TrustedSubnet, "t", "", "trusted subnet")
	cfg.GRPCServer.TrustedSubnet = cfg.Handlers.TrustedSubnet

	flag.StringVar(&cfgFileName, "c", "", "config file")
	flag.Parse()

	// Переменные окружения
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
	if _, envset := os.LookupEnv("ENABLE_HTTPS"); envset {
		cfg.Handlers.EnableHTTPS = true
	}
	if envtrust := os.Getenv("TRUSTED_SUBNET"); envtrust != "" {
		cfg.Handlers.TrustedSubnet = envtrust
		cfg.GRPCServer.TrustedSubnet = envtrust
	}

	if envconfig := os.Getenv("CONFIG"); envconfig != "" {
		cfgFileName = envconfig
	}

	// Файл
	if cfgFileName != "" {
		getConfigFile(&cfg, cfgFileName)
	}

	/* 	if cfg.Repository.DBDsn == "" {
		cfg.Repository.DBDsn = "host=localhost user=bob password=bob dbname=shortener sslmode=disable"
	} */
	if cfg.Repository.DBDsn != "" {
		cfg.Repository.StoreType = repositoryConfig.StoreTypeDB
	} else if cfg.Repository.Filename != "" {
		cfg.Repository.StoreType = repositoryConfig.StoreTypeFile
	} else {
		cfg.Repository.StoreType = repositoryConfig.StoreTypeVar
	}

	// костыль для кривых данных
	cfg.Handlers.ServerAddr = strings.TrimPrefix(cfg.Handlers.ServerAddr, "http://")
	cfg.Handlers.ServerAddr = strings.TrimPrefix(cfg.Handlers.ServerAddr, "http//")
	cfg.Handlers.BaseAddr = strings.TrimPrefix(cfg.Handlers.BaseAddr, "http://")
	cfg.Handlers.BaseAddr = strings.TrimPrefix(cfg.Handlers.BaseAddr, "http//")

	return cfg
}

// ConfigJSON JSON-конфигурация (файл)
type ConfigJSON struct {
	ServerAddress   string `json:"server_address"`
	BaseURL         string `json:"base_url"`
	FileStoragePath string `json:"file_storage_path"`
	DatabaseDSN     string `json:"database_dsn"`
	EnableHTTPS     bool   `json:"enable_https"`
	TrustedSubnet   string `json:"trusted_subnet"`
}

// getConfigFile получить конфигурацию из JSON-файла
func getConfigFile(cfg *Config, file string) {
	f, err := os.Open(file)
	if err != nil {
		return
	}
	defer f.Close()

	var buf bytes.Buffer
	_, err = buf.ReadFrom(f)
	if err != nil {
		return
	}

	var cfgJSON ConfigJSON
	err = json.Unmarshal(buf.Bytes(), &cfgJSON)
	if err != nil {
		return
	}

	if cfg.Handlers.ServerAddr == "" {
		cfg.Handlers.ServerAddr = cfgJSON.ServerAddress
	}
	if cfg.Handlers.BaseAddr == "" {
		cfg.Handlers.BaseAddr = cfgJSON.BaseURL
	}
	if !cfg.Handlers.EnableHTTPS {
		cfg.Handlers.EnableHTTPS = cfgJSON.EnableHTTPS
	}
	if cfg.Repository.Filename == "" {
		cfg.Repository.Filename = cfgJSON.FileStoragePath
	}
	if cfg.Repository.DBDsn == "" {
		cfg.Repository.DBDsn = cfgJSON.DatabaseDSN
	}
	if cfg.Handlers.TrustedSubnet == "" {
		cfg.Handlers.TrustedSubnet = cfgJSON.TrustedSubnet
		cfg.GRPCServer.TrustedSubnet = cfgJSON.TrustedSubnet
	}
}
