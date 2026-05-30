package grpc

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "pz1.2/proto/auth"
	"pz1.2/services/auth/internal/service"
)

type Server struct {
	pb.UnimplementedAuthServiceServer
	authService *service.AuthService
	log         *zap.Logger
}

func NewServer(authService *service.AuthService, log *zap.Logger) *Server {
	return &Server{
		authService: authService,
		log:         log.With(zap.String("component", "grpc_server")),
	}
}

func (s *Server) Verify(ctx context.Context, req *pb.VerifyRequest) (*pb.VerifyResponse, error) {
	s.log.Info("verify request received", zap.Bool("has_auth", req.Token != ""))

	resp, err := s.authService.Verify(req.Token)
	if err != nil {
		s.log.Warn("token verification failed")
		return &pb.VerifyResponse{
			Valid: false,
			Error: "unauthorized",
		}, status.Error(codes.Unauthenticated, "invalid token")
	}

	s.log.Info("token verified", zap.String("subject", resp.Subject))
	return &pb.VerifyResponse{
		Valid:   resp.Valid,
		Subject: resp.Subject,
	}, nil
}

func RegisterServer(s *grpc.Server, authService *service.AuthService, log *zap.Logger) {
	pb.RegisterAuthServiceServer(s, NewServer(authService, log))
}
