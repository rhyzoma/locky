package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rhyzoma/locky/internal/config"
	"github.com/rhyzoma/locky/internal/domain"
	"github.com/rhyzoma/locky/internal/notification"
	"github.com/rhyzoma/locky/internal/repository/memory"
	"github.com/rhyzoma/locky/internal/repository/postgres"
	"github.com/rhyzoma/locky/internal/security"
	"github.com/rhyzoma/locky/internal/transport/grpcserver"
	"github.com/rhyzoma/locky/internal/usecase"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	if err := run(); err != nil {
		slog.Error("locky stopped with an error", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(env("CONFIG_FILE", "configs/locky.local.yaml"))
	if err != nil {
		return err
	}

	users, credentialsStore, sessions, verificationTokens, outbox, closeStore, err := newStores(cfg)
	if err != nil {
		return err
	}
	defer closeStore()

	var mailer domain.Mailer = memory.Mailer{}
	if cfg.Mail.Enabled {
		mailer, err = notification.NewSMTPMailer(cfg.Mail)
		if err != nil {
			return err
		}
	}

	service, err := usecase.NewService(usecase.ServiceConfig{
		Users:       users,
		Credentials: credentialsStore,
		Sessions:    sessions,
		Outbox:      outbox,
		Mailer:      mailer,
		Passwords:   security.BcryptPasswords{},
		Tokens: security.JWT{
			Key:      []byte(cfg.Security.JWTSigningKey),
			KeyID:    cfg.Security.JWTKeyID,
			Issuer:   "locky",
			Audience: "locky",
		},
		VerificationTokens: verificationTokens,
		AutoVerifyEmail:    cfg.Locky.AutoVerifyEmail,
	})
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", cfg.Server.GRPCAddr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer func() {
		if err := listener.Close(); err != nil {
			slog.Warn("close listener", "err", err)
		}
	}()

	tlsConfig, err := security.ServerTLSConfig(cfg.Server.TLS)
	if err != nil {
		return err
	}
	serverOptions := make([]grpc.ServerOption, 0, 2)
	rateLimiter := security.NewRateLimiter(cfg.Security.LoginRateLimit, time.Duration(cfg.Security.LoginRateWindow))
	serverOptions = append(serverOptions, grpc.ChainUnaryInterceptor(
		grpcserver.RecoveryUnaryInterceptor,
		rateLimiter.UnaryInterceptor(nil),
	))
	if tlsConfig != nil {
		serverOptions = append(serverOptions, grpc.Creds(credentials.NewTLS(tlsConfig)))
	}
	server := grpc.NewServer(serverOptions...)
	grpcserver.RegisterService(server, service)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(server, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("locky.v1.LockyService", grpc_health_v1.HealthCheckResponse_SERVING)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
		healthServer.SetServingStatus("locky.v1.LockyService", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
		stopGRPCServer(server)
	}()
	slog.Info("locky service listening", "address", listener.Addr().String())
	if err := server.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}

func stopGRPCServer(server *grpc.Server) {
	const shutdownTimeout = 15 * time.Second

	stopped := make(chan struct{})
	go func() {
		server.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-time.After(shutdownTimeout):
		slog.Warn("graceful shutdown timed out; forcing gRPC server stop", "timeout", shutdownTimeout)
		server.Stop()
	}
}

func newStores(cfg config.Config) (domain.UserRepository, domain.CredentialRepository, domain.SessionStore, domain.VerificationTokenStore, domain.OutboxStore, func(), error) {
	if cfg.Database.DSN == "" {
		store := memory.NewStore()
		return store.Users(), store.Credentials(), store.Sessions(), store.VerificationTokens(), store, func() {}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := postgres.NewPool(ctx, cfg.Database.DSN, cfg.Database.MaxConns)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	store := postgres.New(pool)
	return store.Users(), store.Credentials(), store.Sessions(), store.VerificationTokens(), store.Outbox(), store.Close, nil
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
