package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	authgrpc "pz1.2/services/auth/internal/grpc"
	authhttp "pz1.2/services/auth/internal/http"
	"pz1.2/services/auth/internal/service"
	"pz1.2/shared/logger"
	"pz1.2/shared/middleware"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	log := logger.New("auth")
	defer log.Sync()

	httpPort := os.Getenv("AUTH_PORT")
	if httpPort == "" {
		httpPort = "8081"
	}

	grpcPort := os.Getenv("AUTH_GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051"
	}

	authService := service.NewAuthService()

	mux := http.NewServeMux()
	handler := authhttp.NewHandler(authService, log)
	handler.RegisterRoutes(mux)

	httpHandler := middleware.RequestID(middleware.AccessLog(log)(mux))

	httpServer := &http.Server{
		Addr:         ":" + httpPort,
		Handler:      httpHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	grpcServer := grpc.NewServer()
	authgrpc.RegisterServer(grpcServer, authService, log)

	go func() {
		lis, err := net.Listen("tcp", ":"+grpcPort)
		if err != nil {
			log.Fatal("failed to listen on gRPC port", zap.String("port", grpcPort), zap.Error(err))
		}
		log.Info("gRPC server starting", zap.String("port", grpcPort))
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal("gRPC server failed", zap.Error(err))
		}
	}()

	go func() {
		log.Info("HTTP server starting", zap.String("port", httpPort))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down servers")

	grpcServer.GracefulStop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatal("HTTP server shutdown failed", zap.Error(err))
	}

	log.Info("servers stopped")
}
