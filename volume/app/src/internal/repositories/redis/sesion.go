package redis

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	modelsRedis "api-gateway/internal/models/redis"
)

// SesionRepository gestiona el almacenamiento y operaciones de sesión activa en Redis.
type SesionRepository struct {
	*BaseRepository[modelsRedis.Sesion]
}

// NewSesionRepository inicializa y retorna una nueva instancia de SesionRepository.
func NewSesionRepository(client *goredis.Client) *SesionRepository {
	return &SesionRepository{
		BaseRepository: NewBaseRepository[modelsRedis.Sesion](client),
	}
}

// Save almacena una sesión de usuario en Redis calculando su clave canónica y asignando su tiempo de vida (TTL).
func (r *SesionRepository) Save(ctx context.Context, sesion *modelsRedis.Sesion, timestamp int64, ttl time.Duration) error {
	if sesion == nil {
		return errors.New("la sesión no puede ser nula")
	}
	if sesion.Usuario.ID == uuid.Nil {
		return errors.New("el identificador de usuario en la sesión es inválido")
	}

	key := sesion.BuildKey(timestamp)
	return r.Set(ctx, key, *sesion, ttl)
}

// GetByUsuarioIDAndTimestamp obtiene la sesión de Redis para un usuario y marca de tiempo específicos.
func (r *SesionRepository) GetByUsuarioIDAndTimestamp(ctx context.Context, usuarioID uuid.UUID, timestamp int64) (*modelsRedis.Sesion, error) {
	if usuarioID == uuid.Nil {
		return nil, errors.New("el identificador de usuario es inválido")
	}

	key := modelsRedis.BuildSesionKey(usuarioID, timestamp)
	return r.Get(ctx, key)
}

// DeleteByUsuarioIDAndTimestamp elimina la clave de sesión específica en Redis.
func (r *SesionRepository) DeleteByUsuarioIDAndTimestamp(ctx context.Context, usuarioID uuid.UUID, timestamp int64) error {
	if usuarioID == uuid.Nil {
		return errors.New("el identificador de usuario es inválido")
	}

	key := modelsRedis.BuildSesionKey(usuarioID, timestamp)
	return r.Delete(ctx, key)
}

// DeleteAllByUsuarioID revoca y elimina de forma masiva todas las sesiones activas asociadas a un usuario.
func (r *SesionRepository) DeleteAllByUsuarioID(ctx context.Context, usuarioID uuid.UUID) error {
	if usuarioID == uuid.Nil {
		return errors.New("el identificador de usuario es inválido")
	}

	pattern := fmt.Sprintf("%s:%s:*", modelsRedis.SesionKeyPrefix, usuarioID.String())
	return r.DeleteByPattern(ctx, pattern)
}

// ListKeysByUsuarioID busca y retorna todas las claves de sesión activas para un usuario.
func (r *SesionRepository) ListKeysByUsuarioID(ctx context.Context, usuarioID uuid.UUID) ([]string, error) {
	if usuarioID == uuid.Nil {
		return nil, errors.New("el identificador de usuario es inválido")
	}

	pattern := fmt.Sprintf("%s:%s:*", modelsRedis.SesionKeyPrefix, usuarioID.String())
	return r.FindKeysByPattern(ctx, pattern)
}

// RenewSesionTTL extiende el tiempo de vida (TTL) de una sesión activa específica.
func (r *SesionRepository) RenewSesionTTL(ctx context.Context, usuarioID uuid.UUID, timestamp int64, ttl time.Duration) error {
	if usuarioID == uuid.Nil {
		return errors.New("el identificador de usuario es inválido")
	}

	key := modelsRedis.BuildSesionKey(usuarioID, timestamp)
	return r.RenewTTL(ctx, key, ttl)
}

// GetBySesionKey obtiene la sesión de Redis a partir de su clave canónica sesion:usuario_id:timestamp.
func (r *SesionRepository) GetBySesionKey(ctx context.Context, key string) (*modelsRedis.Sesion, error) {
	if strings.TrimSpace(key) == "" {
		return nil, errors.New("la clave de sesión no puede estar vacía")
	}

	return r.Get(ctx, key)
}
