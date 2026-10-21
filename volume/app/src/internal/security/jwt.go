package security

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWT administra la generación y validación de tokens JSON Web Token (JWT).
type JWT struct {
	secretKey []byte
}

// Claims contiene la clave de sesión y los reclamos estándar registrados del token JWT.
type Claims struct {
	SesionKey string `json:"sesion_key"`
	jwt.RegisteredClaims
}

// NewJWT inicializa una nueva instancia de JWT con la clave secreta provista.
func NewJWT(secretKey string) *JWT {
	return &JWT{
		secretKey: []byte(secretKey),
	}
}

// Generate genera un token JWT firmado con algoritmo HS256 incluyendo la clave de sesión y tiempo de expiración.
func (j *JWT) Generate(sesionKey string, ttl time.Duration) (string, error) {
	if sesionKey == "" {
		return "", errors.New("la clave de sesión no puede estar vacía")
	}

	now := time.Now()

	claims := Claims{
		SesionKey: sesionKey,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(j.secretKey)
	if err != nil {
		return "", fmt.Errorf("error al firmar el token: %w", err)
	}

	return tokenString, nil
}

// Validate analiza y verifica la firma y vigencia de un token JWT retornando sus claims si es válido.
func (j *JWT) Validate(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, errors.New("el token no puede estar vacío")
	}

	var claims Claims
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("método de firma inesperado: %v", t.Header["alg"])
		}

		return j.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("token inválido")
	}

	return &claims, nil
}
