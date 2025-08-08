package config

import (
    "log"
    "os"
    "strconv"

    "github.com/joho/godotenv"
)

type Config struct {
	TelegramBotToken string
	WebAddress       string
	CertPath         string
	KeyPath          string
	DataPath         string
    BackupTime       string // HH:MM local time
    BackupTimezone   string // e.g., Europe/Moscow
    BackupRetention  int    // days to keep backups
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
        BackupTime:       getEnv("BACKUP_TIME", "03:00"),
        BackupTimezone:   getEnv("BACKUP_TIMEZONE", ""),
        BackupRetention:  getEnvInt("BACKUP_RETENTION_DAYS", 30),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
    if value, ok := os.LookupEnv(key); ok {
        if n, err := strconv.Atoi(value); err == nil {
            return n
        }
    }
    return fallback
}
