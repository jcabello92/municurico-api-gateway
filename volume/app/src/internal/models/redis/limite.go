package redis

import (
	"fmt"
	"time"
)

// LimiteKeyPrefix define el prefijo base utilizado para las claves de rate limit en Redis.
const LimiteKeyPrefix = "limite"

// BuildLimiteIPKey genera la clave estandarizada de Redis para el rate limit por dirección IP.
func BuildLimiteIPKey(ip string) string {
	return fmt.Sprintf("%s:ip:%s", LimiteKeyPrefix, ip)
}

// Limite representa el resultado de la evaluación del control de tasa de peticiones.
type Limite struct {
	Identificador string        `json:"identificador"`
	Maximo        int64         `json:"maximo"`
	Restantes     int64         `json:"restantes"`
	Reinicio      time.Duration `json:"reinicio"`
	Excedido      bool          `json:"excedido"`
}
