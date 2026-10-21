package clients

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "api-gateway/proto"
)

// AccesoClient administra la conexión gRPC y el cliente del servicio de acceso hacia core-service.
type AccesoClient struct {
	conn   *grpc.ClientConn
	Client pb.AccesoServiceClient
}

// NewAccesoClient inicializa y establece una nueva conexión gRPC para el servicio de acceso hacia core-service.
func NewAccesoClient(host, port string) (*AccesoClient, error) {
	target := fmt.Sprintf("%s:%s", host, port)

	conn, err := grpc.NewClient(
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("fallo al conectar con core-service en %s: %w", target, err)
	}

	return &AccesoClient{
		conn:   conn,
		Client: pb.NewAccesoServiceClient(conn),
	}, nil
}

// Close cierra la conexión gRPC subyacente con core-service.
func (c *AccesoClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
