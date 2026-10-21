package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"

	"api-gateway/internal/clients"
	"api-gateway/internal/response"
)

// HealthHandler gestiona las comprobaciones de estado de salud del api-gateway y sus dependencias.
type HealthHandler struct {
	redisClient  *redis.Client
	healthClient *clients.HealthClient
}

// NewHealthHandler inicializa y retorna una nueva instancia de HealthHandler.
func NewHealthHandler(redisClient *redis.Client, healthClient *clients.HealthClient) *HealthHandler {
	return &HealthHandler{
		redisClient:  redisClient,
		healthClient: healthClient,
	}
}

// Check responde con el estado de operatividad del api-gateway, Redis y servicios gRPC dependientes.
func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	redisStatus := "connected"
	if err := h.redisClient.Ping(ctx).Err(); err != nil {
		redisStatus = "disconnected"
	}

	coreStatus := "serving"
	if h.healthClient != nil {
		status, err := h.healthClient.Check(ctx, "")
		if err != nil || status != "serving" {
			coreStatus = status
		}
	} else {
		coreStatus = "not_configured"
	}

	services := map[string]any{
		"redis":        redisStatus,
		"core_service": coreStatus,
	}

	if redisStatus != "connected" || coreStatus != "serving" {
		response.Error(w, http.StatusServiceUnavailable, "Servicio no disponible", "servicio parcialmente no disponible", map[string]any{
			"status":   "degraded",
			"services": services,
		})
		return
	}

	response.Success(w, http.StatusOK, "Estado del servicio", "api-gateway operativo", map[string]any{
		"status":   "ok",
		"services": services,
	})
}
