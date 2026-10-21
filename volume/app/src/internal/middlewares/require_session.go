package middlewares

import (
	"context"
	"net/http"
	"strings"

	"google.golang.org/grpc/metadata"

	modelsRedis "api-gateway/internal/models/redis"
	reposRedis "api-gateway/internal/repositories/redis"
	"api-gateway/internal/response"
	"api-gateway/internal/security"
)

const (
	sesionContextKey    contextKey = "usuario_sesion"
	sesionKeyContextKey contextKey = "usuario_sesion_key"
	autorIDContextKey   contextKey = "usuario_autor_id"
	tokenContextKey     contextKey = "usuario_token"
)

// RequireSession valida que el token JWT sea válido y exista en Redis, inyectando la sesión en el contexto de la petición.
func RequireSession(jwtValidator *security.JWT, sesionRepo *reposRedis.SesionRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := ExtractToken(r)
			if tokenString == "" {
				response.Error(w, http.StatusUnauthorized, "No autenticado", "no autenticado: falta token de sesión", nil)
				return
			}

			if jwtValidator == nil || sesionRepo == nil {
				response.Error(w, http.StatusInternalServerError, "Error de servidor", "error interno de configuración de autenticación", nil)
				return
			}

			// 1. Validación de firma criptográfica y vigencia del JWT
			claims, err := jwtValidator.Validate(tokenString)
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "No autenticado", "no autenticado: token inválido o expirado", nil)
				return
			}

			if strings.TrimSpace(claims.SesionKey) == "" {
				response.Error(w, http.StatusUnauthorized, "No autenticado", "no autenticado: el token no contiene clave de sesión", nil)
				return
			}

			// 2. Validación de existencia de la sesión activa en Redis
			sesion, err := sesionRepo.GetBySesionKey(r.Context(), claims.SesionKey)
			if err != nil || sesion == nil {
				response.Error(w, http.StatusUnauthorized, "No autenticado", "no autenticado: la sesión ha expirado o no existe", nil)
				return
			}

			autorID := sesion.Usuario.ID.String()

			// 3. Inyección de la sesión e identificadores en el contexto HTTP
			ctx := r.Context()
			ctx = context.WithValue(ctx, sesionContextKey, sesion)
			ctx = context.WithValue(ctx, sesionKeyContextKey, claims.SesionKey)
			ctx = context.WithValue(ctx, autorIDContextKey, autorID)
			ctx = context.WithValue(ctx, tokenContextKey, tokenString)

			// 4. Inyección de x-autor-id verificado en metadatos salientes gRPC
			ctx = metadata.AppendToOutgoingContext(ctx, "x-autor-id", autorID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ExtractToken obtiene el token JWT desde la cabecera Authorization (Bearer) o desde la cookie HttpOnly 'token'.
func ExtractToken(r *http.Request) string {
	if r == nil {
		return ""
	}

	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	}

	if cookie, err := r.Cookie("token"); err == nil && strings.TrimSpace(cookie.Value) != "" {
		return strings.TrimSpace(cookie.Value)
	}

	return ""
}

// GetSesionFromContext obtiene los datos de la sesión deserializada almacenada en el contexto.
func GetSesionFromContext(ctx context.Context) *modelsRedis.Sesion {
	if ctx == nil {
		return nil
	}
	if sesion, ok := ctx.Value(sesionContextKey).(*modelsRedis.Sesion); ok {
		return sesion
	}
	return nil
}

// GetSesionKeyFromContext obtiene la clave canónica de sesión almacenada en el contexto.
func GetSesionKeyFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if key, ok := ctx.Value(sesionKeyContextKey).(string); ok {
		return key
	}
	return ""
}

// GetAutorIDFromContext obtiene el identificador de usuario verificado almacenado en el contexto.
func GetAutorIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(autorIDContextKey).(string); ok {
		return id
	}
	return ""
}

// GetTokenFromContext obtiene el token JWT en crudo almacenado en el contexto.
func GetTokenFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if token, ok := ctx.Value(tokenContextKey).(string); ok {
		return token
	}
	return ""
}
