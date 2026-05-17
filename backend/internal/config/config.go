package config

import "os"

// Config содержит все внешние настройки backend-процесса: адрес HTTP API,
// строку подключения к PostgreSQL, адрес симулятора и каталог XLSX-выгрузок.
type Config struct {
	HTTPAddr     string
	DatabaseURL  string
	SimulatorURL string
	ExportDir    string
}

// Load собирает конфигурацию из переменных окружения и подставляет локальные
// значения по умолчанию, подходящие для запуска без Docker.
func Load() Config {
	return Config{
		HTTPAddr:     env("HTTP_ADDR", ":8080"),
		DatabaseURL:  env("DATABASE_URL", "postgres://powertemp:powertemp@localhost:5432/powertemp?sslmode=disable"),
		SimulatorURL: env("SIMULATOR_URL", "http://localhost:8090"),
		ExportDir:    env("EXPORT_DIR", "exports"),
	}
}

// env возвращает значение переменной окружения или fallback, если она не задана.
func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
