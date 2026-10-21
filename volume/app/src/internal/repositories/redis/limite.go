package redis

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"

	modelsRedis "api-gateway/internal/models/redis"
)

// LimiteRepository gestiona el contador y expiración del control de tasa de peticiones en Redis.
type LimiteRepository struct {
	*BaseRepository[modelsRedis.Limite]
}

// NewLimiteRepository inicializa y retorna una nueva instancia de LimiteRepository.
func NewLimiteRepository(client *goredis.Client) *LimiteRepository {
	return &LimiteRepository{
		BaseRepository: NewBaseRepository[modelsRedis.Limite](client),
	}
}

// VerificarIP incrementa y evalúa el contador de peticiones para una dirección IP en la ventana de tiempo dada.
func (r *LimiteRepository) VerificarIP(ctx context.Context, ip string, maximo int64, ventana time.Duration) (*modelsRedis.Limite, error) {
	if strings.TrimSpace(ip) == "" {
		return nil, errors.New("la dirección ip no puede estar vacía")
	}
	if maximo <= 0 {
		return nil, errors.New("el límite máximo de peticiones debe ser mayor a cero")
	}
	if ventana <= 0 {
		return nil, errors.New("la ventana de tiempo debe ser mayor a cero")
	}

	key := modelsRedis.BuildLimiteIPKey(ip)

	// Incremento atómico del contador
	contador, err := r.Increment(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("error al incrementar contador de límite: %w", err)
	}

	// Si es la primera petición de la ventana, se fija la expiración
	if contador == 1 {
		if err := r.Expire(ctx, key, ventana); err != nil {
			return nil, fmt.Errorf("error al asignar ventana de tiempo al límite: %w", err)
		}
	}

	// Consulta del tiempo restante de la ventana actual
	ttl, err := r.TTL(ctx, key)
	if err != nil || ttl < 0 {
		ttl = ventana
	}

	restantes := maximo - contador
	if restantes < 0 {
		restantes = 0
	}

	return &modelsRedis.Limite{
		Identificador: ip,
		Maximo:        maximo,
		Restantes:     restantes,
		Reinicio:      ttl,
		Excedido:      contador > maximo,
	}, nil
}
