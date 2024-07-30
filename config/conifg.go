package config

import (
	"os"
)

type Config struct {
	DatabaseDSN string
	RedisAddr   string
	GRPCPort    string
	HTTPPort    string
}

func LoadConfig() *Config {
	return &Config{
		DatabaseDSN: getEnv("DATABASE_DSN", "postgresql://postgres:admin@127.0.0.1:5432/tugas_shorturl"),
		RedisAddr:   getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		GRPCPort:    getEnv("GRPC_PORT", ":50051"),
		HTTPPort:    getEnv("HTTP_PORT", ":8080"),
	}
}

func getEnv(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}
