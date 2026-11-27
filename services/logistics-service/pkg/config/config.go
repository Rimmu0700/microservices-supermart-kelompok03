package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv        string
	AppPort       string
	MySQLHost     string
	MySQLPort     string
	MySQLDatabase string
	MySQLUser     string
	MySQLPassword string
	JWTSecret     string
	JWTExpiry     string
	CORSOrigins   []string
}

func LoadConfig() *Config {
	_ = godotenv.Load()

	appEnv := getEnv("APP_ENV", "development")
	appPort := getEnv("APP_PORT", "3000")
	mysqlHost := getEnv("MYSQL_HOST", "127.0.0.1")
	mysqlPort := getEnv("MYSQL_PORT", "3306")
	mysqlDatabase := getEnv("MYSQL_DATABASE", "nusantara_db")
	mysqlUser := getEnv("MYSQL_USER", "nusantara_user")
	mysqlPassword := getEnv("MYSQL_PASSWORD", "nusantara_secret")
	jwtSecret := getEnv("JWT_SECRET", "supermart-secret-key-for-nusantara-course-2026")
	jwtExpiry := getEnv("JWT_EXPIRY", "24h")
	corsRaw := getEnv("CORS_ORIGINS", "http://localhost:4200,http://127.0.0.1:4200,http://localhost:3000")

	corsOrigins := strings.Split(corsRaw, ",")
	for i := range corsOrigins {
		corsOrigins[i] = strings.TrimSpace(corsOrigins[i])
	}

	return &Config{
		AppEnv:        appEnv,
		AppPort:       appPort,
		MySQLHost:     mysqlHost,
		MySQLPort:     mysqlPort,
		MySQLDatabase: mysqlDatabase,
		MySQLUser:     mysqlUser,
		MySQLPassword: mysqlPassword,
		JWTSecret:     jwtSecret,
		JWTExpiry:     jwtExpiry,
		CORSOrigins:   corsOrigins,
	}
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}
