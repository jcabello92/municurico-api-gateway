package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config centraliza los valores de configuración y variables de entorno del api-gateway.
type Config struct {
	// Servicio
	AppEnv   string
	HTTPHost string
	HTTPPort string

	// Core Service (gRPC)
	CoreServiceHost string
	CoreServicePort string

	// Redis
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int

	// Security
	JWTSecret  string
	SessionTTL time.Duration

	// Rate Limit
	RateLimitRequests int
	RateLimitWindow   time.Duration
}

// LoadEnv carga y valida la configuración del sistema para el api-gateway.
func LoadEnv() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("ADVERTENCIA: No se encontró el archivo .env. Se intentarán cargar las variables de entorno desde el sistema.")
	}

	cfg := &Config{
		// Servicio
		AppEnv:   GetEnvOrDefault("APP_ENV", "prod"),
		HTTPHost: GetEnvOrDefault("HTTP_HOST", "0.0.0.0"),
		HTTPPort: GetEnvOrDefault("HTTP_PORT", "8080"),

		// Core Service (gRPC)
		CoreServiceHost: GetEnvOrCrash("CORE_SERVICE_HOST"),
		CoreServicePort: GetEnvOrDefault("CORE_SERVICE_PORT", "50051"),

		// Redis
		RedisHost:     GetEnvOrCrash("REDIS_HOST"),
		RedisPort:     GetEnvOrDefault("REDIS_PORT", "6379"),
		RedisPassword: GetEnvOrCrash("REDIS_PASSWORD"),
		RedisDB:       GetEnvAsIntOrDefault("REDIS_DB", 0),

		// Security
		JWTSecret:  GetEnvOrCrash("JWT_SECRET"),
		SessionTTL: GetEnvAsDurationOrDefault("SESSION_TTL", time.Hour),

		// Rate Limit
		RateLimitRequests: GetEnvAsIntOrDefault("RATE_LIMIT_REQUESTS", 60),
		RateLimitWindow:   GetEnvAsDurationOrDefault("RATE_LIMIT_WINDOW", time.Minute),
	}

	return cfg
}

// GetEnvOrDefault busca una variable de entorno (string); si no existe, devuelve un valor por defecto.
func GetEnvOrDefault(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return fallback
}

// GetEnvOrCrash busca una variable de entorno (string); si no existe, aborta la ejecución.
func GetEnvOrCrash(key string) string {
	value, exists := os.LookupEnv(key)
	if !exists || value == "" {
		log.Fatalf("ERROR CRÍTICO: La variable de entorno '%s' es obligatoria y no está definida.", key)
	}
	return value
}

// GetEnvAsIntOrDefault busca una variable de entorno (int); si no existe, devuelve un valor por defecto.
func GetEnvAsIntOrDefault(key string, fallback int) int {
	valueStr, exists := os.LookupEnv(key)
	if !exists || valueStr == "" {
		return fallback
	}
	val, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Fatalf("ERROR CRÍTICO: La variable de entorno '%s' debe ser un número entero válido.", key)
	}
	return val
}

// GetEnvAsDurationOrDefault busca una variable de entorno y la convierte a time.Duration; si no existe, devuelve el valor por defecto.
func GetEnvAsDurationOrDefault(key string, fallback time.Duration) time.Duration {
	valueStr, exists := os.LookupEnv(key)
	if !exists || valueStr == "" {
		return fallback
	}

	d, err := time.ParseDuration(valueStr)
	if err == nil {
		return d
	}

	if seconds, err := strconv.Atoi(valueStr); err == nil {
		return time.Duration(seconds) * time.Second
	}

	log.Fatalf("ERROR CRÍTICO: La variable de entorno '%s' debe tener un formato de duración válido (ej. 1h, 60m, 3600s).", key)
	return fallback
}
