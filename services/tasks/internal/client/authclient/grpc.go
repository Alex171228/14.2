package authclient

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	pb "pz1.2/proto/auth"
	"pz1.2/shared/middleware"
)

type GRPCClient struct {
	conn    *grpc.ClientConn
	client  pb.AuthServiceClient
	timeout time.Duration
	log     *zap.Logger
}

func NewGRPCClient(addr string, timeout time.Duration, log *zap.Logger) (*GRPCClient, error) {
	conn, err := grpc.Dial(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to auth service: %w", err)
	}

	return &GRPCClient{
		conn:    conn,
		client:  pb.NewAuthServiceClient(conn),
		timeout: timeout,
		log:     log.With(zap.String("component", "auth_client_grpc")),
	}, nil
}

func (c *GRPCClient) Verify(ctx context.Context, token string) (*VerifyResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	requestID := middleware.GetRequestID(ctx)
	l := c.log.With(zap.String("request_id", requestID))
	l.Debug("calling auth gRPC verify")

	resp, err := c.client.Verify(ctx, &pb.VerifyRequest{Token: token})
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.Unauthenticated {
			l.Warn("auth gRPC verify: unauthorized")
			return &VerifyResponse{
				Valid: false,
				Error: "unauthorized",
			}, nil
		}
		l.Error("auth gRPC verify failed", zap.Error(err))
		return nil, fmt.Errorf("auth service error: %w", err)
	}

	l.Debug("auth gRPC verify: success", zap.String("subject", resp.Subject))
	return &VerifyResponse{
		Valid:   resp.Valid,
		Subject: resp.Subject,
	}, nil
}

func (c *GRPCClient) Close() error {
	return c.conn.Close()
}
