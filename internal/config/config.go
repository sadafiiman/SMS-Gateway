package config

import (
	"os"
	"strconv"
	"time"

	"github.com/iman/sms-gateway/internal/domain"
)

type Config struct {
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration

	NormalWorkers  int
	ExpressWorkers int
	QueueSize      int

	Prices domain.PriceList
}

func Load() Config {
	return Config{
		Port:            getEnv("PORT", "8080"),
		ReadTimeout:     getEnvDuration("READ_TIMEOUT", 5*time.Second),
		WriteTimeout:    getEnvDuration("WRITE_TIMEOUT", 10*time.Second),
		ShutdownTimeout: getEnvDuration("SHUTDOWN_TIMEOUT", 15*time.Second),

		NormalWorkers:  getEnvInt("NORMAL_WORKERS", 20),
		ExpressWorkers: getEnvInt("EXPRESS_WORKERS", 10),
		QueueSize:      getEnvInt("QUEUE_SIZE", 10000),

		Prices: domain.PriceList{
			Normal:  int64(getEnvInt("PRICE_NORMAL", 100)),
			OTP:     int64(getEnvInt("PRICE_OTP", 150)),
			Express: int64(getEnvInt("PRICE_EXPRESS", 300)),
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
