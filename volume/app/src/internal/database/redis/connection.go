package redis

import (
	"context"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
)

// BuildRedisDSN genera una cadena de conexión para Redis.
func BuildRedisDSN(host, port string) string {
	return fmt.Sprintf("%s:%s", host, port)
}

// ConnectRedis conecta a la base de datos Redis.
func ConnectRedis(dsn string, password string, db int) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     dsn,
		Password: password,
		DB:       db,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("ERROR CRÍTICO: No se pudo conectar a Redis: %v", err)
	}

	log.Println("Conexión a Redis establecida exitosamente.")
	return client
}
