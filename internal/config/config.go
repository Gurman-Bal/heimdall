package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DBPath                 string
	DefaultLogDir          string
	APIAddr                string
	OllamaURL              string
	LLMModel               string
	ReportInterval         time.Duration
	AuthUsername           string
	AuthPassword           string
	SelfContainer          string
	ControllableContainers []string
	SessionTimeout         time.Duration
	ActivityRetention      time.Duration
	EventBufferSize        int
	BatchSize              int
	BatchFlushInterval     time.Duration
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

	cfg.AuthUsername = getEnv("HEIMDALL_AUTH_USER", "admin")
	cfg.AuthPassword = getEnv("HEIMDALL_AUTH_PASS", "")
	cfg.SelfContainer = getEnv("HEIMDALL_CONTAINER_NAME", "heimdall")
	cfg.ControllableContainers = strings.Split(getEnv("HEIMDALL_CONTROLLABLE_CONTAINERS", "heimdall,heimdall-ollama"), ",")

	cfg.SessionTimeout = getDuration("HEIMDALL_SESSION_TIMEOUT", 30*time.Minute)
	cfg.ActivityRetention = getDuration("HEIMDALL_ACTIVITY_RETENTION", 48*time.Hour)
	cfg.ReportInterval = getDuration("HEIMDALL_REPORT_INTERVAL", time.Hour)
	cfg.BatchFlushInterval = getDuration("HEIMDALL_BATCH_FLUSH_INTERVAL", 500*time.Millisecond)

	cfg.EventBufferSize = getInt("HEIMDALL_EVENT_BUFFER_SIZE", 5000)
	cfg.BatchSize = getInt("HEIMDALL_BATCH_SIZE", 500)

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		slog.Warn("invalid duration env var, using default", "key", key, "value", v, "default", fallback)
		return fallback
	}
	return d
}

func getInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	var n int
	if _, err := fmt.Sscan(v, &n); err != nil || n <= 0 {
		slog.Warn("invalid int env var, using default", "key", key, "value", v, "default", fallback)
		return fallback
	}
	return n
}
