package tests

import (
	"TemplatestPGSQL/internal/repo/mocks"
	"TemplatestPGSQL/internal/service"
	pb "TemplatestPGSQL/pkg/auf"
	"context"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"testing"
)

const testSecret = "his name is Toooken"

func TestAuthService_Register(t *testing.T) {
	tests := []struct {
		name          string
		req           *pb.RegisterRequest
		mockSetup     func(*mocks.Repository)
		expectedError bool
		expectedCode  codes.Code
	}{
		{
			name: "successful registration",
			req: &pb.RegisterRequest{
				Username: "testuser",
				Password: "password123",
			},
			mockSetup: func(mockRepo *mocks.Repository) {
				mockRepo.On("CreateUser", mock.Anything, "testuser", mock.AnythingOfType("string")).
					Return(nil)
			},
			expectedError: false,
		},
		{
			name: "username already exists",
			req: &pb.RegisterRequest{
				Username: "existinguser",
				Password: "password123",
			},
			mockSetup: func(mockRepo *mocks.Repository) {
				pgErr := &pgconn.PgError{Code: "23505"}
				mockRepo.On("CreateUser", mock.Anything, "existinguser", mock.AnythingOfType("string")).
					Return(pgErr)
			},
			expectedError: true,
			expectedCode:  codes.AlreadyExists,
		},
		{
			name: "internal database error",
			req: &pb.RegisterRequest{
				Username: "testuser",
				Password: "password123",
			},
			mockSetup: func(mockRepo *mocks.Repository) {
				mockRepo.On("CreateUser", mock.Anything, "testuser", mock.AnythingOfType("string")).
					Return(errors.New("db connection failed"))
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

			resp, err := authService.Register(context.Background(), tt.req)

			if tt.expectedError {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok, "error should be a grpc status error")
				assert.Equal(t, tt.expectedCode, st.Code())
			} else {
				require.NoError(t, err)
				assert.Equal(t, "User testuser created successfully", resp.Message)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
