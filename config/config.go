package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramBotToken string
	WebAddress       string
	CertPath         string
	KeyPath          string
	DataPath         string
}

func Load() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using environment variables")
	}

	return &Config{
		TelegramBotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
		WebAddress:       getEnv("WEB_ADDRESS", "0.0.0.0:8088"),
		CertPath:         getEnv("CERT_PATH", ""),
		KeyPath:          getEnv("KEY_PATH", ""),
		DataPath:         getEnv("DATA_PATH", "/app/data/data.csv"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
