package response

import (
	"encoding/json"
	"net/http"
)

// Response representa la estructura estándar para todas las respuestas JSON del api-gateway.
type Response struct {
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Title   string `json:"title,omitempty"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Errors  any    `json:"errors,omitempty"`
}

// JSON escribe una respuesta con formato JSON y código de estado en el ResponseWriter.
func JSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

// Success envía una respuesta JSON exitosa con código de estado, título, mensaje y datos en el ResponseWriter.
func Success(w http.ResponseWriter, statusCode int, title string, message string, data any) {
	JSON(w, statusCode, Response{
		Success: true,
		Code:    statusCode,
		Title:   resolveTitle(title, statusCode),
		Message: message,
		Data:    data,
	})
}

// Error envía una respuesta JSON de error con código de estado, título, mensaje y detalles en el ResponseWriter.
func Error(w http.ResponseWriter, statusCode int, title string, message string, errDetails any) {
	JSON(w, statusCode, Response{
		Success: false,
		Code:    statusCode,
		Title:   resolveTitle(title, statusCode),
		Message: message,
		Errors:  errDetails,
	})
}

// resolveTitle determina el título final de la respuesta utilizando un valor por defecto si el provisto está vacío.
func resolveTitle(title string, statusCode int) string {
	if title != "" {
		return title
	}
	return defaultTitle(statusCode)
}

// defaultTitle retorna un título descriptivo en español asociado al código de estado HTTP provisto.
func defaultTitle(statusCode int) string {
	switch statusCode {
	case http.StatusOK:
		return "Operación exitosa"
	case http.StatusCreated:
		return "Recurso creado"
	case http.StatusAccepted:
		return "Solicitud aceptada"
	case http.StatusNoContent:
		return "Sin contenido"
	case http.StatusBadRequest:
		return "Solicitud inválida"
	case http.StatusUnauthorized:
		return "No autenticado"
	case http.StatusForbidden:
		return "Acceso denegado"
	case http.StatusNotFound:
		return "Recurso no encontrado"
	case http.StatusConflict:
		return "Conflicto"
	case http.StatusTooManyRequests:
		return "Límite excedido"
	case http.StatusInternalServerError:
		return "Error interno del servidor"
	case http.StatusBadGateway:
		return "Error de comunicación"
	case http.StatusServiceUnavailable:
		return "Servicio no disponible"
	case http.StatusGatewayTimeout:
		return "Tiempo de espera agotado"
	default:
		if statusCode >= 200 && statusCode < 300 {
			return "Operación exitosa"
		}
		if statusCode >= 400 && statusCode < 500 {
			return "Error de solicitud"
		}
		return "Error del servidor"
	}
}
