package grpcserver

import (
	"context"

	lockyv1 "github.com/rhyzoma/locky/api/gen/locky/v1"
	"github.com/rhyzoma/locky/internal/domain"
	"google.golang.org/grpc"
)

type Service interface {
	Register(context.Context, string, string) (domain.User, error)
	Login(context.Context, string, string) (domain.TokenPair, domain.User, error)
	Refresh(context.Context, string) (domain.TokenPair, error)
	Logout(context.Context, string) error
	VerifyEmail(context.Context, string) error
	CurrentUser(context.Context, string) (domain.User, error)
}

type Server struct {
	lockyv1.UnimplementedLockyServiceServer
	service Service
}

var _ lockyv1.LockyServiceServer = (*Server)(nil)

func New(service Service) *Server {
	return &Server{service: service}
}

func RegisterService(server *grpc.Server, service Service) {
	lockyv1.RegisterLockyServiceServer(server, New(service))
}
