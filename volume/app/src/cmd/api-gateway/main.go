package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"api-gateway/internal/clients"
	"api-gateway/internal/config"
	dbRedis "api-gateway/internal/database/redis"
	"api-gateway/internal/handlers"
	reposRedis "api-gateway/internal/repositories/redis"
	"api-gateway/internal/router"
	"api-gateway/internal/security"
)

func main() {
	cfg := config.LoadEnv()
	log.Printf("Iniciando api-gateway en modo: %s", cfg.AppEnv)

	// 1. Conexión a Redis
	redisDSN := dbRedis.BuildRedisDSN(cfg.RedisHost, cfg.RedisPort)
	redisClient := dbRedis.ConnectRedis(redisDSN, cfg.RedisPassword, cfg.RedisDB)
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Printf("ADVERTENCIA: Error al cerrar cliente Redis: %v.", err)
		}
	}()

	// 2. Conexión gRPC con servicios backend
	healthClient, err := clients.NewHealthClient(cfg.CoreServiceHost, cfg.CoreServicePort)
	if err != nil {
		log.Fatalf("ERROR CRÍTICO: No se pudo conectar con el servicio de salud de core-service: %v.", err)
	}
	defer func() {
		if err := healthClient.Close(); err != nil {
			log.Printf("ADVERTENCIA: Error al cerrar cliente gRPC de salud: %v.", err)
		}
	}()

	accesoClient, err := clients.NewAccesoClient(cfg.CoreServiceHost, cfg.CoreServicePort)
	if err != nil {
		log.Fatalf("ERROR CRÍTICO: No se pudo conectar con core-service (acceso): %v.", err)
	}
	defer func() {
		if err := accesoClient.Close(); err != nil {
			log.Printf("ADVERTENCIA: Error al cerrar conexión gRPC con core-service (acceso): %v.", err)
		}
	}()

	// 3. Inicialización de repositorios
	sesionRepo := reposRedis.NewSesionRepository(redisClient)
	limiteRepo := reposRedis.NewLimiteRepository(redisClient)

	// 4. Inicialización de seguridad y utilidades
	jwtValidator := security.NewJWT(cfg.JWTSecret)

	// 5. Inicialización de handlers
	healthHandler := handlers.NewHealthHandler(redisClient, healthClient)
	accesoHandler := handlers.NewAccesoHandler(accesoClient, sesionRepo, cfg.SessionTTL, cfg.AppEnv == "prod")

	// 6. Configuración de rutas y middlewares
	httpHandler := router.NewRouter(router.RouterDeps{
		HealthHandler:     healthHandler,
		AccesoHandler:     accesoHandler,
		AccesoClient:      accesoClient,
		SesionRepo:        sesionRepo,
		LimiteRepo:        limiteRepo,
		JWT:               jwtValidator,
		RateLimitRequests: cfg.RateLimitRequests,
		RateLimitWindow:   cfg.RateLimitWindow,
	})

	// 6. Servidor HTTP
	addr := fmt.Sprintf("%s:%s", cfg.HTTPHost, cfg.HTTPPort)
	server := &http.Server{
		Addr:         addr,
		Handler:      httpHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Servidor HTTP escuchando en http://%s...", addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("ERROR CRÍTICO: Fallo en ejecución del servidor HTTP: %v.", err)
		}
	}()

	// 7. Espera de señales de apagado limpio (graceful shutdown)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop
	log.Println("Deteniendo servidor HTTP limpiamente...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("ADVERTENCIA: Cierre forzado del servidor HTTP: %v.", err)
	}

	log.Println("Servicio api-gateway detenido con éxito.")
}
