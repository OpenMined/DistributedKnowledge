package config

import (
	"os"
	"strconv"
)

// Config holds application configuration settings
type Config struct {
	// Server settings
	ServerAddr string
	// Rate limiting settings
	MessageRateLimit  float64 // messages per second per user
	MessageBurstLimit int     // maximum burst size
}

// GetEnv returns the value of the environment variable or a default value.
func GetEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}

// GetEnvFloat returns the value of the environment variable as a float64 or a default value.
func GetEnvFloat(key string, defaultVal float64) float64 {
	if value, exists := os.LookupEnv(key); exists {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
	}
	return defaultVal
}

// GetEnvInt returns the value of the environment variable as an int or a default value.
func GetEnvInt(key string, defaultVal int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultVal
}

// LoadConfig loads the application configuration from environment variables.
func LoadConfig() *Config {
	return &Config{
		ServerAddr:        GetEnv("SERVER_ADDR", ":443"),
		MessageRateLimit:  GetEnvFloat("MESSAGE_RATE_LIMIT", 5.0), // 5 messages per second by default
		MessageBurstLimit: GetEnvInt("MESSAGE_BURST_LIMIT", 10),   // burst of 10 messages by default
	}
}
