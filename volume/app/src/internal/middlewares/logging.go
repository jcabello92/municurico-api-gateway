package middlewares

import (
	"log"
	"net/http"
	"time"
)

// statusResponseWriter captura el código de estado HTTP escrito para los logs.
type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader intercepta y almacena el código de estado HTTP.
func (rw *statusResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Logging intercepta las peticiones HTTP para registrar el método, la ruta y la duración de respuesta.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &statusResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(wrapped, r)

		log.Printf("%s %s %d %s", r.Method, r.URL.Path, wrapped.statusCode, time.Since(start))
	})
}
