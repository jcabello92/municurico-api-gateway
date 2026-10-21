package router

import (
	"net/http"
	"time"

	"api-gateway/internal/clients"
	"api-gateway/internal/handlers"
	"api-gateway/internal/middlewares"
	reposRedis "api-gateway/internal/repositories/redis"
	"api-gateway/internal/security"
)

// RouterDeps contiene las dependencias requeridas para registrar las rutas HTTP.
type RouterDeps struct {
	HealthHandler     *handlers.HealthHandler
	AccesoHandler     *handlers.AccesoHandler
	AccesoClient      *clients.AccesoClient
	SesionRepo        *reposRedis.SesionRepository
	LimiteRepo        *reposRedis.LimiteRepository
	JWT               *security.JWT
	RateLimitRequests int
	RateLimitWindow   time.Duration
}

// NewRouter inicializa y configura el árbol de rutas HTTP del api-gateway junto con sus middlewares globales.
func NewRouter(deps RouterDeps) http.Handler {
	mux := http.NewServeMux()

	// Rutas de salud y diagnóstico
	mux.HandleFunc("GET /health", deps.HealthHandler.Check)

	// Rutas del servicio de acceso
	if deps.AccesoHandler != nil {
		proteger := middlewares.RequireSession(deps.JWT, deps.SesionRepo)

		// Rutas públicas de acceso (no requieren sesión)
		mux.HandleFunc("POST /api/v1/acceso/login", deps.AccesoHandler.Login)
		mux.HandleFunc("POST /api/v1/acceso/forgot-password", deps.AccesoHandler.ForgotPassword)
		mux.HandleFunc("POST /api/v1/acceso/reset-password", deps.AccesoHandler.ResetPassword)

		// Rutas privadas protegidas por sesión activa
		mux.Handle("GET /api/v1/acceso/me", proteger(http.HandlerFunc(deps.AccesoHandler.Me)))
		mux.Handle("POST /api/v1/acceso/logout", proteger(http.HandlerFunc(deps.AccesoHandler.Logout)))
		mux.Handle("POST /api/v1/acceso/logout-all", proteger(http.HandlerFunc(deps.AccesoHandler.LogoutAll)))
		mux.Handle("POST /api/v1/acceso/update-password", proteger(http.HandlerFunc(deps.AccesoHandler.UpdatePassword)))

		// Rutas privadas con permiso administrativo específico
		permisoRegistro := middlewares.RequirePermission("acceso:registro", "usuarios:crear")
		mux.Handle("POST /api/v1/acceso/register", proteger(permisoRegistro(http.HandlerFunc(deps.AccesoHandler.Register))))
	}

	// Aplicación de middlewares globales en cadena
	var handler http.Handler = mux
	handler = middlewares.Auditoria(handler)
	if deps.LimiteRepo != nil && deps.RateLimitRequests > 0 {
		handler = middlewares.RateLimit(deps.LimiteRepo, int64(deps.RateLimitRequests), deps.RateLimitWindow)(handler)
	}
	handler = middlewares.CORS(handler)
	handler = middlewares.Logging(handler)
	handler = middlewares.Recovery(handler)

	return handler
}
