package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"api-gateway/internal/clients"
	"api-gateway/internal/middlewares"
	reposRedis "api-gateway/internal/repositories/redis"
	"api-gateway/internal/response"
	pb "api-gateway/proto"
)

type loginRequest struct {
	Run               *string `json:"run"`
	Pasaporte         *string `json:"pasaporte"`
	CorreoElectronico *string `json:"correo_electronico"`
	Contrasena        string  `json:"contrasena"`
}

type updatePasswordRequest struct {
	ContrasenaActual string `json:"contrasena_actual"`
	NuevaContrasena  string `json:"nueva_contrasena"`
}

type registerRequest struct {
	Run               *string `json:"run"`
	Pasaporte         *string `json:"pasaporte"`
	Nombre            string  `json:"nombre"`
	ApellidoPaterno   *string `json:"apellido_paterno"`
	ApellidoMaterno   *string `json:"apellido_materno"`
	Genero            *string `json:"genero"`
	FechaNacimiento   *string `json:"fecha_nacimiento"`
	ComunaID          *string `json:"comuna_id"`
	Direccion         *string `json:"direccion"`
	TelefonoFijo      *string `json:"telefono_fijo"`
	TelefonoMovil     *string `json:"telefono_movil"`
	CorreoElectronico string  `json:"correo_electronico"`
	Contrasena        string  `json:"contrasena"`
}

type forgotPasswordRequest struct {
	CorreoElectronico string `json:"correo_electronico"`
}

type resetPasswordRequest struct {
	Token           string `json:"token"`
	NuevaContrasena string `json:"nueva_contrasena"`
}

// AccesoHandler gestiona las peticiones HTTP de autenticación y sesión traduciéndolas a llamadas gRPC hacia core-service.
type AccesoHandler struct {
	accesoClient *clients.AccesoClient
	sesionRepo   *reposRedis.SesionRepository
	sessionTTL   time.Duration
	isProd       bool
}

// NewAccesoHandler inicializa y retorna una nueva instancia de AccesoHandler.
func NewAccesoHandler(
	accesoClient *clients.AccesoClient,
	sesionRepo *reposRedis.SesionRepository,
	sessionTTL time.Duration,
	isProd bool,
) *AccesoHandler {
	return &AccesoHandler{
		accesoClient: accesoClient,
		sesionRepo:   sesionRepo,
		sessionTTL:   sessionTTL,
		isProd:       isProd,
	}
}

// Login atiende la autenticación de usuarios y emite la sesión y credenciales correspondientes.
func (h *AccesoHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Solicitud inválida", "formato de solicitud inválido", nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	resp, err := h.accesoClient.Client.IniciarSesion(ctx, &pb.IniciarSesionRequest{
		Run:               req.Run,
		Pasaporte:         req.Pasaporte,
		CorreoElectronico: req.CorreoElectronico,
		Contrasena:        req.Contrasena,
	})
	if err != nil {
		statusHTTP, title, msg := grpcErrorToHTTP(err)
		response.Error(w, statusHTTP, title, msg, nil)
		return
	}

	platform := middlewares.GetPlatformFromContext(r.Context())
	if platform == "mobile" {
		response.Success(w, http.StatusOK, "Inicio de sesión", "sesión iniciada exitosamente", resp)
		return
	}

	// Web y Desktop: envuelve el token en cookie HttpOnly segura
	h.setTokenCookie(w, resp.GetToken(), int(resp.GetExpiresIn()))
	response.Success(w, http.StatusOK, "Inicio de sesión", "sesión iniciada exitosamente", map[string]any{
		"token_type": resp.GetTokenType(),
		"expires_in": resp.GetExpiresIn(),
		"sesion":     resp.GetSesion(),
	})
}

// Logout procesa el cierre de la sesión activa del usuario y revoca sus credenciales.
func (h *AccesoHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token := h.extractToken(r)
	if token == "" {
		response.Error(w, http.StatusUnauthorized, "No autenticado", "no autenticado: falta token de sesión", nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	_, err := h.accesoClient.Client.CerrarSesion(ctx, &pb.CerrarSesionRequest{
		Token: token,
	})
	if err != nil {
		statusHTTP, title, msg := grpcErrorToHTTP(err)
		response.Error(w, statusHTTP, title, msg, nil)
		return
	}

	h.clearTokenCookie(w)
	response.Success(w, http.StatusOK, "Cierre de sesión", "sesión cerrada exitosamente", nil)
}

// LogoutAll revoca todas las sesiones activas asociadas al usuario autenticado.
func (h *AccesoHandler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	token := h.extractToken(r)
	if token == "" {
		response.Error(w, http.StatusUnauthorized, "No autenticado", "no autenticado: falta token de sesión", nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	_, err := h.accesoClient.Client.CerrarTodasLasSesiones(ctx, &pb.CerrarTodasLasSesionesRequest{
		Token: token,
	})
	if err != nil {
		statusHTTP, title, msg := grpcErrorToHTTP(err)
		response.Error(w, statusHTTP, title, msg, nil)
		return
	}

	h.clearTokenCookie(w)
	response.Success(w, http.StatusOK, "Cierre de sesiones", "todas las sesiones activas han sido revocadas exitosamente", nil)
}

// Me retorna la información de perfil y sesión activa del usuario autenticado recuperada desde la caché.
func (h *AccesoHandler) Me(w http.ResponseWriter, r *http.Request) {
	sesion := middlewares.GetSesionFromContext(r.Context())
	if sesion == nil {
		response.Error(w, http.StatusUnauthorized, "No autenticado", "no autenticado: sesión no encontrada", nil)
		return
	}

	response.Success(w, http.StatusOK, "Perfil de usuario", "perfil recuperado exitosamente", sesion)
}

// UpdatePassword gestiona el cambio seguro de credenciales del usuario autenticado.
func (h *AccesoHandler) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	token := h.extractToken(r)
	if token == "" {
		response.Error(w, http.StatusUnauthorized, "No autenticado", "no autenticado: falta token de sesión", nil)
		return
	}

	var req updatePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Solicitud inválida", "formato de solicitud inválido", nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	resp, err := h.accesoClient.Client.ActualizarContrasena(ctx, &pb.ActualizarContrasenaRequest{
		Token:            token,
		ContrasenaActual: req.ContrasenaActual,
		NuevaContrasena:  req.NuevaContrasena,
	})
	if err != nil {
		statusHTTP, title, msg := grpcErrorToHTTP(err)
		response.Error(w, statusHTTP, title, msg, nil)
		return
	}

	response.Success(w, http.StatusOK, "Actualización de contraseña", resp.GetMensaje(), nil)
}

// Register atiende el autoregistro de una nueva persona y cuenta de usuario.
func (h *AccesoHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Solicitud inválida", "formato de solicitud inválido", nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	resp, err := h.accesoClient.Client.RegistrarUsuario(ctx, &pb.RegistrarUsuarioRequest{
		Run:               req.Run,
		Pasaporte:         req.Pasaporte,
		Nombre:            req.Nombre,
		ApellidoPaterno:   req.ApellidoPaterno,
		ApellidoMaterno:   req.ApellidoMaterno,
		Genero:            req.Genero,
		FechaNacimiento:   req.FechaNacimiento,
		ComunaId:          req.ComunaID,
		Direccion:         req.Direccion,
		TelefonoFijo:      req.TelefonoFijo,
		TelefonoMovil:     req.TelefonoMovil,
		CorreoElectronico: req.CorreoElectronico,
		Contrasena:        req.Contrasena,
	})
	if err != nil {
		statusHTTP, title, msg := grpcErrorToHTTP(err)
		response.Error(w, statusHTTP, title, msg, nil)
		return
	}

	response.Success(w, http.StatusCreated, "Registro de usuario", resp.GetMensaje(), map[string]string{
		"usuario_id": resp.GetUsuarioId(),
		"persona_id": resp.GetPersonaId(),
	})
}

// ForgotPassword procesa la solicitud de emisión de código para restablecimiento de contraseña.
func (h *AccesoHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Solicitud inválida", "formato de solicitud inválido", nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	resp, err := h.accesoClient.Client.SolicitarRecuperacionContrasena(ctx, &pb.SolicitarRecuperacionContrasenaRequest{
		CorreoElectronico: req.CorreoElectronico,
	})
	if err != nil {
		statusHTTP, title, msg := grpcErrorToHTTP(err)
		response.Error(w, statusHTTP, title, msg, nil)
		return
	}

	response.Success(w, http.StatusOK, "Recuperación de contraseña", resp.GetMensaje(), nil)
}

// ResetPassword valida el código de recuperación y asigna la nueva contraseña del usuario.
func (h *AccesoHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Solicitud inválida", "formato de solicitud inválido", nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	resp, err := h.accesoClient.Client.RestablecerContrasena(ctx, &pb.RestablecerContrasenaRequest{
		Token:           req.Token,
		NuevaContrasena: req.NuevaContrasena,
	})
	if err != nil {
		statusHTTP, title, msg := grpcErrorToHTTP(err)
		response.Error(w, statusHTTP, title, msg, nil)
		return
	}

	response.Success(w, http.StatusOK, "Restablecimiento de contraseña", resp.GetMensaje(), nil)
}

// extractToken obtiene el token de sesión desde la cabecera Authorization o desde la cookie.
func (h *AccesoHandler) extractToken(r *http.Request) string {
	return middlewares.ExtractToken(r)
}

// setTokenCookie configura la cookie de sesión con opciones de seguridad en el ResponseWriter.
func (h *AccesoHandler) setTokenCookie(w http.ResponseWriter, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(time.Duration(maxAge) * time.Second),
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   h.isProd,
		SameSite: http.SameSiteStrictMode,
	})
}

// clearTokenCookie invalida la cookie de sesión en el ResponseWriter.
func (h *AccesoHandler) clearTokenCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.isProd,
		SameSite: http.SameSiteStrictMode,
	})
}

// grpcErrorToHTTP traduce los códigos de estado y errores de gRPC a códigos de respuesta HTTP, título y mensaje descriptivo.
func grpcErrorToHTTP(err error) (int, string, string) {
	st, ok := status.FromError(err)
	if !ok {
		return http.StatusInternalServerError, "Error interno", "error interno del servidor"
	}

	switch st.Code() {
	case codes.InvalidArgument:
		return http.StatusBadRequest, "Solicitud inválida", st.Message()
	case codes.Unauthenticated:
		return http.StatusUnauthorized, "No autenticado", st.Message()
	case codes.PermissionDenied:
		return http.StatusForbidden, "Acceso denegado", st.Message()
	case codes.NotFound:
		return http.StatusNotFound, "Recurso no encontrado", st.Message()
	case codes.AlreadyExists:
		return http.StatusConflict, "Conflicto", st.Message()
	case codes.Unavailable:
		return http.StatusServiceUnavailable, "Servicio no disponible", "servicio backend no disponible"
	default:
		return http.StatusInternalServerError, "Error interno", st.Message()
	}
}
