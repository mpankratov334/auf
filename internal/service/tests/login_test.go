package tests

import (
	repo2 "TemplatestPGSQL/internal/repo"
	"TemplatestPGSQL/internal/repo/mocks"
	"TemplatestPGSQL/internal/service"
	pb "TemplatestPGSQL/pkg/auf"
	"context"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"testing"
)

func TestAuthService_Login(t *testing.T) {
	validPassword := "correct_password"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(validPassword), bcrypt.DefaultCost)

	tests := []struct {
		name          string
		req           *pb.LoginRequest
		mockSetup     func(*mocks.Repository)
		expectedError bool
		expectedCode  codes.Code
		validateToken func(string) bool
	}{
		{
			name: "successful login",
			req: &pb.LoginRequest{
				Username: "testuser",
				Password: validPassword,
			},
			mockSetup: func(mockRepo *mocks.Repository) {
				mockRepo.On("GetUser", mock.Anything, "testuser").
					Return(&repo2.User{
						Username:     "testuser",
						PasswordHash: string(hashedPassword),
					}, nil)
			},
			expectedError: false,
			validateToken: func(token string) bool {
				_, err := jwt.ParseWithClaims(
					token,
					&jwt.RegisteredClaims{},
					func(token *jwt.Token) (interface{}, error) {
						return []byte(testSecret), nil
					},
				)
				return err == nil
			},
		},
		{
			name: "user not found",
			req: &pb.LoginRequest{
				Username: "unknownuser",
				Password: validPassword,
			},
			mockSetup: func(mockRepo *mocks.Repository) {
				mockRepo.On("GetUser", mock.Anything, "unknownuser").
					Return(nil, pgx.ErrNoRows)
			},
			expectedError: true,
			expectedCode:  codes.NotFound,
		},
		{
			name: "invalid password",
			req: &pb.LoginRequest{
				Username: "testuser",
				Password: "wrong_password",
			},
			mockSetup: func(mockRepo *mocks.Repository) {
				mockRepo.On("GetUser", mock.Anything, "testuser").
					Return(&repo2.User{
						Username:     "testuser",
						PasswordHash: string(hashedPassword),
					}, nil)
			},
			expectedError: true,
			expectedCode:  codes.Unauthenticated,
		},
		{
			name: "database error",
			req: &pb.LoginRequest{
				Username: "testuser",
				Password: validPassword,
			},
			mockSetup: func(mockRepo *mocks.Repository) {
				mockRepo.On("GetUser", mock.Anything, "testuser").
					Return(nil, errors.New("db error"))
			},
			expectedError: true,
			expectedCode:  codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(mocks.Repository)
			tt.mockSetup(mockRepo)

			logger := zap.NewNop().Sugar()
			authService := service.AuthService{
				Repo: mockRepo,
				Log:  logger,
			}

			resp, err := authService.Login(context.Background(), tt.req)

			if tt.expectedError {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok, "error should be a grpc status error")
				assert.Equal(t, tt.expectedCode, st.Code())
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, resp.Token, "token should not be empty")

				// Validate the token structure if validateToken is provided
				if tt.validateToken != nil {
					assert.True(t, tt.validateToken(resp.Token), "generated token should be valid")
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
