# api-gateway — Ecosistema municurico

Servicio API Gateway que actúa como terminador HTTP/REST hacia los clientes (Web, Mobile, Desktop), gestiona sesiones en Redis y traduce solicitudes hacia `core-service` mediante gRPC.

## Requisitos previos

Crear la red externa de Docker definida en `.env` (si aún no existe):

```bash
docker network create municurico-network
```

## Uso en desarrollo

1. Iniciar el contenedor:
   ```bash
   docker compose up -d
   ```

2. El contenedor ejecuta hot-reload automático mediante **Air** sobre el código montado en `./volume/app/src/`.

## Uso en producción

Compilar y levantar en modo producción:

```bash
docker compose -f docker-compose.prod.yml up -d --build
```
