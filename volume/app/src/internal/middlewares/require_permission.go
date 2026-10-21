package middlewares

import (
	"net/http"

	"api-gateway/internal/response"
)

// RequirePermission valida que la sesión autenticada en el contexto contenga al menos uno de los permisos requeridos o rol SUPERADMIN.
func RequirePermission(permisos ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sesion := GetSesionFromContext(r.Context())
			if sesion == nil {
				response.Error(w, http.StatusUnauthorized, "No autenticado", "no autenticado: falta sesión de usuario", nil)
				return
			}

			if !sesion.TieneAlgunPermiso(permisos...) {
				response.Error(w, http.StatusForbidden, "Acceso denegado", "acceso denegado: permisos insuficientes para realizar esta acción", nil)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
