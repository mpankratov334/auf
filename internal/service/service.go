package service

import (
	"TemplatestPGSQL/internal/config"
	repo2 "TemplatestPGSQL/internal/repo"
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net"

	pb "TemplatestPGSQL/pkg/auf"
)

type AuthService struct {
	pb.UnimplementedAuthServiceServer
	repo     repo2.Repository
	log      *zap.SugaredLogger
	Service  *grpc.Server
	listener net.Listener
}

func NewService(repo repo2.Repository, logger *zap.SugaredLogger, cfg config.GRPC) *AuthService {
	lis, err := net.Listen("tcp", ":"+cfg.Port)
	if err != nil {
		logger.Fatalf("NewService failed to listen: %v", err)
	}
	server := grpc.NewServer()
	auf := &AuthService{
		repo:     repo,
		log:      logger,
		Service:  server,
		listener: lis,
	}
	pb.RegisterAuthServiceServer(server, auf)
	return auf
}

func (s *AuthService) ListenAndServe() error {
	err := s.Service.Serve(s.listener)
	return err
}

func (s *AuthService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	// sha hash
	// repo post
	// send  pb response
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to hash password: %v", err)
	}

	err = s.repo.CreateUser(ctx, req.Username, string(hashedPassword))
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			// Код ошибки unique_violation
			return nil, status.Errorf(codes.AlreadyExists, "username %s already exists", req.Username)
		}
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}

	return &pb.RegisterResponse{
		Message: fmt.Sprintf("User %s created successfully", req.Username),
	}, nil
}

func (s *AuthService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	user, err := s.repo.GetUser(ctx, req.Username)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "user %s not found", req.Username)
		}
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid password")
	}

	token := "his name is Token"

	return &pb.LoginResponse{
		Token: token,
	}, nil
}
