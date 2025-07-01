package service

import (
	"TemplatestPGSQL/internal/config"
	repo2 "TemplatestPGSQL/internal/repo"
	pb "TemplatestPGSQL/pkg/auf"
	"context"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net"
	"time"
)

var secretKey = []byte("his name is Toooken")

type AuthService struct {
	pb.UnimplementedAuthServiceServer
	Repo     repo2.Repository
	Log      *zap.SugaredLogger
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
		Repo:     repo,
		Log:      logger,
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
	// send pb response
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to hash password: %v", err)
	}

	err = s.Repo.CreateUser(ctx, req.Username, string(hashedPassword))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
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
	user, err := s.Repo.GetUser(ctx, req.Username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "user %s not found", req.Username)
		}
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid password")
	}

	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to sign token: %v", err)
	}

	return &pb.LoginResponse{
		Token: tokenString,
	}, nil
}

func (s *AuthService) ValidateToken(ctx context.Context, req *pb.Token) (*pb.IsValid, error) {
	if req.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "Token is required")
	}
	token, err := jwt.ParseWithClaims(
		req.Token,
		&jwt.RegisteredClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return secretKey, nil
		},
	)

	if err != nil || !token.Valid {
		return &pb.IsValid{IsValid: false}, nil
	}
	return &pb.IsValid{IsValid: true}, nil
}
