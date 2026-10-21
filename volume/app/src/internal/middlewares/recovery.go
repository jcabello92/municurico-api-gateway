package middlewares

import (
	"log"
	"net/http"

	"api-gateway/internal/response"
)

// Recovery captura fallos inesperados y retorna una respuesta JSON de error interno del servidor.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("ERROR CRÍTICO: Pánico no controlado en petición HTTP: %v.", rec)
				response.Error(w, http.StatusInternalServerError, "Error interno", "error interno del servidor", nil)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
