package config

import (
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DBPath         string
	DefaultLogDir  string
	APIAddr        string
	OllamaURL      string
	LLMModel       string
	ReportInterval time.Duration
}

func Load() Config {
	if err := godotenv.Load(); err != nil {
		slog.Info(".env file not found, using environment/defaults")
	}

	cfg := Config{
		DBPath:        getEnv("HEIMDALL_DB_PATH", "./heimdall.db"),
		DefaultLogDir: getEnv("HEIMDALL_LOG_DIR", "./testlogs"),
		APIAddr:       getEnv("HEIMDALL_API_ADDR", ":8080"),
		OllamaURL:     getEnv("HEIMDALL_OLLAMA_URL", "http://localhost:11434"),
		LLMModel:      getEnv("HEIMDALL_LLM_MODEL", "qwen2.5:0.5b"),
	}

	interval := getEnv("HEIMDALL_REPORT_INTERVAL", "1h")
	d, err := time.ParseDuration(interval)
	if err != nil {
		slog.Warn("invalid HEIMDALL_REPORT_INTERVAL, using default", "value", interval, "default", "1h")
		d = time.Hour
	}
	cfg.ReportInterval = d

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
