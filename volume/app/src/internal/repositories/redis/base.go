package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// ErrKeyNotFound indica que la clave solicitada no existe o ha expirado en Redis.
var ErrKeyNotFound = goredis.Nil

// BaseRepository define la estructura base genérica para operaciones CRUD y utilidades en Redis.
type BaseRepository[T any] struct {
	client *goredis.Client
}

// NewBaseRepository inicializa y retorna una nueva instancia de BaseRepository para el tipo genérico T.
func NewBaseRepository[T any](client *goredis.Client) *BaseRepository[T] {
	return &BaseRepository[T]{client: client}
}

// Set serializa en formato JSON y almacena un valor en Redis con un tiempo de vida (TTL) determinado.
func (r *BaseRepository[T]) Set(ctx context.Context, key string, value T, ttl time.Duration) error {
	if strings.TrimSpace(key) == "" {
		return errors.New("la clave de redis no puede estar vacía")
	}

	bytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("error al serializar valor para redis: %w", err)
	}

	if err := r.client.Set(ctx, key, bytes, ttl).Err(); err != nil {
		return fmt.Errorf("error al guardar en redis con clave '%s': %w", key, err)
	}

	return nil
}

// Get recupera y deserializa un valor almacenado en Redis según su clave.
func (r *BaseRepository[T]) Get(ctx context.Context, key string) (*T, error) {
	if strings.TrimSpace(key) == "" {
		return nil, errors.New("la clave de redis no puede estar vacía")
	}

	bytes, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return nil, ErrKeyNotFound
		}
		return nil, fmt.Errorf("error al obtener clave '%s' de redis: %w", key, err)
	}

	var data T
	if err := json.Unmarshal(bytes, &data); err != nil {
		return nil, fmt.Errorf("error al deserializar valor de redis para clave '%s': %w", key, err)
	}

	return &data, nil
}

// Exists verifica si una clave existe actualmente en Redis.
func (r *BaseRepository[T]) Exists(ctx context.Context, key string) (bool, error) {
	if strings.TrimSpace(key) == "" {
		return false, errors.New("la clave de redis no puede estar vacía")
	}

	n, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("error al verificar existencia de clave '%s' en redis: %w", key, err)
	}

	return n > 0, nil
}

// TTL consulta el tiempo de expiración restante de una clave en Redis.
func (r *BaseRepository[T]) TTL(ctx context.Context, key string) (time.Duration, error) {
	if strings.TrimSpace(key) == "" {
		return 0, errors.New("la clave de redis no puede estar vacía")
	}

	duration, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("error al consultar ttl de clave '%s' en redis: %w", key, err)
	}

	return duration, nil
}

// Update sobreescribe el valor en Redis conservando el tiempo de expiración (TTL) original.
func (r *BaseRepository[T]) Update(ctx context.Context, key string, value T) error {
	if strings.TrimSpace(key) == "" {
		return errors.New("la clave de redis no puede estar vacía")
	}

	bytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("error al serializar valor para redis: %w", err)
	}

	if err := r.client.Set(ctx, key, bytes, goredis.KeepTTL).Err(); err != nil {
		return fmt.Errorf("error al actualizar clave '%s' en redis: %w", key, err)
	}

	return nil
}

// RenewTTL renueva el tiempo de vida (TTL) de una clave existente en Redis.
func (r *BaseRepository[T]) RenewTTL(ctx context.Context, key string, ttl time.Duration) error {
	if strings.TrimSpace(key) == "" {
		return errors.New("la clave de redis no puede estar vacía")
	}

	if err := r.client.Expire(ctx, key, ttl).Err(); err != nil {
		return fmt.Errorf("error al renovar ttl para clave '%s' en redis: %w", key, err)
	}

	return nil
}

// Delete elimina un registro de Redis dada su clave.
func (r *BaseRepository[T]) Delete(ctx context.Context, key string) error {
	if strings.TrimSpace(key) == "" {
		return errors.New("la clave de redis no puede estar vacía")
	}

	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("error al eliminar clave '%s' de redis: %w", key, err)
	}

	return nil
}

// Increment incrementa atómicamente en 1 el contador numérico de la clave y retorna el nuevo valor.
func (r *BaseRepository[T]) Increment(ctx context.Context, key string) (int64, error) {
	if strings.TrimSpace(key) == "" {
		return 0, errors.New("la clave de redis no puede estar vacía")
	}

	val, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("error al incrementar contador en redis para clave '%s': %w", key, err)
	}

	return val, nil
}

// Expire establece un tiempo de expiración (TTL) sobre una clave existente.
func (r *BaseRepository[T]) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if strings.TrimSpace(key) == "" {
		return errors.New("la clave de redis no puede estar vacía")
	}

	if err := r.client.Expire(ctx, key, ttl).Err(); err != nil {
		return fmt.Errorf("error al establecer expiración para clave '%s' en redis: %w", key, err)
	}

	return nil
}

// FindKeysByPattern busca de manera iterativa y no bloqueante todas las claves que coincidan con un patrón.
func (r *BaseRepository[T]) FindKeysByPattern(ctx context.Context, pattern string) ([]string, error) {
	if strings.TrimSpace(pattern) == "" {
		return nil, errors.New("el patrón de búsqueda no puede estar vacío")
	}

	var allKeys []string
	var cursor uint64

	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("error al escanear claves con patrón '%s': %w", pattern, err)
		}

		allKeys = append(allKeys, keys...)
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return allKeys, nil
}

// DeleteByPattern busca y elimina atómicamente todas las claves que coincidan con el patrón proporcionado.
func (r *BaseRepository[T]) DeleteByPattern(ctx context.Context, pattern string) error {
	keys, err := r.FindKeysByPattern(ctx, pattern)
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		return nil
	}

	if err := r.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("error al eliminar masivamente claves con patrón '%s': %w", pattern, err)
	}

	return nil
}

// UpdateByPattern busca y actualiza el valor de todas las claves que coincidan con el patrón, preservando sus TTL individuales.
func (r *BaseRepository[T]) UpdateByPattern(ctx context.Context, pattern string, value T) error {
	keys, err := r.FindKeysByPattern(ctx, pattern)
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		return nil
	}

	bytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("error al serializar valor para actualización masiva en redis: %w", err)
	}

	pipe := r.client.Pipeline()
	for _, key := range keys {
		pipe.Set(ctx, key, bytes, goredis.KeepTTL)
	}

	if _, err = pipe.Exec(ctx); err != nil {
		return fmt.Errorf("error al ejecutar pipeline de actualización en redis para patrón '%s': %w", pattern, err)
	}

	return nil
}

// GetClient retorna el cliente subyacente de Redis para operaciones avanzadas.
func (r *BaseRepository[T]) GetClient() *goredis.Client {
	return r.client
}
