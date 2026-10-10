package grpcserver_test

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	lockyv1 "github.com/rhyzoma/locky/api/gen/locky/v1"
	"github.com/rhyzoma/locky/internal/domain"
	"github.com/rhyzoma/locky/internal/transport/grpcserver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestServerRegisterOverGRPC(t *testing.T) {
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	grpcserver.RegisterService(server, fakeService{})
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.Serve(listener)
	}()
	t.Cleanup(func() {
		server.Stop()
		_ = listener.Close()
		select {
		case err := <-serveErr:
			if err != nil && !errors.Is(err, grpc.ErrServerStopped) {
				t.Errorf("Serve() error = %v", err)
			}
		case <-time.After(time.Second):
			t.Error("Serve() did not stop within one second")
		}
	})

	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("DialContext() error = %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	response, err := lockyv1.NewLockyServiceClient(conn).Register(context.Background(), &lockyv1.RegisterRequest{
		Email:    "user@example.com",
		Password: "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if response.GetUser().GetEmail() != "user@example.com" {
		t.Fatalf("Register() email = %q, want %q", response.GetUser().GetEmail(), "user@example.com")
	}
}

type fakeService struct{}

func (fakeService) Register(context.Context, string, string) (domain.User, error) {
	return domain.User{ID: "user-1", Email: "user@example.com", Status: domain.UserStatusActive}, nil
}

func (fakeService) Login(context.Context, string, string) (domain.TokenPair, domain.User, error) {
	return domain.TokenPair{}, domain.User{}, nil
}

func (fakeService) Refresh(context.Context, string) (domain.TokenPair, error) {
	return domain.TokenPair{}, nil
}

func (fakeService) Logout(context.Context, string) error { return nil }

func (fakeService) VerifyEmail(context.Context, string) error { return nil }

func (fakeService) CurrentUser(context.Context, string) (domain.User, error) {
	return domain.User{}, nil
}
