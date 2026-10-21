package middlewares

import (
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	reposRedis "api-gateway/internal/repositories/redis"
	"api-gateway/internal/response"
)

// RateLimit intercepta las peticiones HTTP y aplica control de tasa por IP excluyendo rutas de diagnóstico.
func RateLimit(limiteRepo *reposRedis.LimiteRepository, maximo int64, ventana time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Exclusión de ruta de diagnóstico y solicitudes preflight de CORS
			if r.URL.Path == "/health" || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			if limiteRepo == nil || maximo <= 0 {
				next.ServeHTTP(w, r)
				return
			}

			ip := extractClientIP(r)

			limite, err := limiteRepo.VerificarIP(r.Context(), ip, maximo, ventana)
			if err != nil {
				log.Printf("ADVERTENCIA: Error al verificar límite de peticiones para ip '%s': %v.", ip, err)
				next.ServeHTTP(w, r)
				return
			}

			reinicioSegundos := int64(math.Ceil(limite.Reinicio.Seconds()))
			if reinicioSegundos < 0 {
				reinicioSegundos = 0
			}

			w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(limite.Maximo, 10))
			w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(limite.Restantes, 10))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(reinicioSegundos, 10))

			if limite.Excedido {
				w.Header().Set("Retry-After", strconv.FormatInt(reinicioSegundos, 10))
				response.Error(w, http.StatusTooManyRequests, "Límite excedido", "límite de peticiones excedido, intente más tarde", nil)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
