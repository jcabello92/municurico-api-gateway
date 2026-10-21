package clients

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// HealthClient administra la conexión gRPC y consultas de salud hacia un servicio remoto.
type HealthClient struct {
	conn   *grpc.ClientConn
	client grpc_health_v1.HealthClient
}

// NewHealthClient inicializa y establece una nueva conexión gRPC para monitoreo de salud.
func NewHealthClient(host, port string) (*HealthClient, error) {
	target := fmt.Sprintf("%s:%s", host, port)

	conn, err := grpc.NewClient(
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("fallo al conectar con servicio gRPC para salud en %s: %w", target, err)
	}

	return &HealthClient{
		conn:   conn,
		client: grpc_health_v1.NewHealthClient(conn),
	}, nil
}

// Check consulta el estado de operatividad de un servicio gRPC mediante el protocolo estándar de salud.
func (c *HealthClient) Check(ctx context.Context, serviceName string) (string, error) {
	resp, err := c.client.Check(ctx, &grpc_health_v1.HealthCheckRequest{
		Service: serviceName,
	})
	if err != nil {
		return "unavailable", err
	}

	if resp.GetStatus() == grpc_health_v1.HealthCheckResponse_SERVING {
		return "serving", nil
	}

	return resp.GetStatus().String(), nil
}

// Close cierra la conexión gRPC subyacente del cliente de salud.
func (c *HealthClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
