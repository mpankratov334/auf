package tests

import (
	"TemplatestPGSQL/internal/service"
	pb "TemplatestPGSQL/pkg/auf"
	"context"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"testing"
	"time"
)

func TestAuthService_ValidateToken(t *testing.T) {
	// Helper function to generate a valid token
	generateValidToken := func() string {
		claims := jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString([]byte(testSecret))
		return tokenString
	}

	tests := []struct {
		name         string
		token        string
		expected     bool
		expectedCode codes.Code
	}{
		{
			name:     "valid token",
			token:    generateValidToken(),
			expected: true,
		},
		{
			name:     "invalid token",
			token:    "invalid.token.string",
			expected: false,
		},
		{
			name:     "expired token",
			token:    generateExpiredToken(),
			expected: false,
		},
		{
			name:         "empty token",
			token:        "",
			expected:     false,
			expectedCode: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zap.NewNop().Sugar()
			authService := service.AuthService{
				Log: logger,
			}

			resp, err := authService.ValidateToken(context.Background(), &pb.Token{Token: tt.token})

			if tt.expectedCode != codes.OK {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok, "error should be a grpc status error")
				assert.Equal(t, tt.expectedCode, st.Code())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, resp.IsValid)
			}
		})
	}
}

// Helper function to generate expired token
func generateExpiredToken() string {
	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-24 * time.Hour)), // Expired 24 hours ago
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(testSecret))
	return tokenString
}
