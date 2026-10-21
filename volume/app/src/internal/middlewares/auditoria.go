package middlewares

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"google.golang.org/grpc/metadata"
)

type contextKey string

const platformContextKey contextKey = "client_platform"

// Auditoria intercepta las peticiones HTTP, extrae metadatos del cliente e inyecta la información en el contexto para llamadas gRPC.
func Auditoria(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Extracción de IP real del cliente
		ip := extractClientIP(r)

		// 2. Extracción de User-Agent
		ua := r.UserAgent()
		if strings.TrimSpace(ua) == "" {
			ua = "desconocido"
		}

		// 3. Extracción de plataforma (web, mobile, desktop)
		platform := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Platform")))
		if platform == "" {
			platform = "web"
		}

		// 4. Extracción de identificador de autor si la petición incluye token
		autorID := extractAutorID(r)

		// 5. Configuración de pares de metadatos para gRPC
		mdPairs := []string{
			"x-autor-ip", ip,
			"x-user-agent", ua,
			"x-platform", platform,
		}
		if autorID != "" {
			mdPairs = append(mdPairs, "x-autor-id", autorID)
		}

		// 6. Inyección de metadatos salientes gRPC y plataforma en el contexto
		ctx := metadata.NewOutgoingContext(r.Context(), metadata.Pairs(mdPairs...))
		ctx = context.WithValue(ctx, platformContextKey, platform)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetPlatformFromContext obtiene la plataforma del cliente almacenada en el contexto.
func GetPlatformFromContext(ctx context.Context) string {
	if ctx == nil {
		return "web"
	}
	if platform, ok := ctx.Value(platformContextKey).(string); ok && platform != "" {
		return platform
	}
	return "web"
}

// extractClientIP obtiene la dirección IP real del cliente evaluando cabeceras de proxy o la dirección remota.
func extractClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return strings.TrimSpace(realIP)
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}

	return r.RemoteAddr
}

// extractAutorID obtiene el identificador de usuario a partir del token JWT si la petición está autenticada.
func extractAutorID(r *http.Request) string {
	token := ExtractToken(r)
	if token == "" {
		return ""
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ""
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}

	var claims struct {
		SesionKey string `json:"sesion_key"`
	}
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return ""
	}

	keyParts := strings.Split(claims.SesionKey, ":")
	if len(keyParts) >= 2 {
		return keyParts[1]
	}

	return ""
}
