package controllers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/adamanr/employes_service/internal/entity"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

var LoginRequestFieldDescriptions = []pgconn.FieldDescription{
	{Name: "email", DataTypeOID: 25},    // text (string)
	{Name: "password", DataTypeOID: 25}, // text (string)
}

func TestAuthController_AuthLogin(t *testing.T) {
	tests := []struct {
		name          string
		loginReq      *entity.LoginRequest
		setupMocks    func(*MockDB, *MockRedis)
		expectError   bool
		errorContains string
	}{
		{
			name: "redis error on access token",
			loginReq: &entity.LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			setupMocks: func(mockDB *MockDB, mockRedis *MockRedis) {
				hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
				passwordStr := string(hashedPassword)

				mockRow := NewMockRow([]interface{}{
					uint64(1), "test@example.com", passwordStr, "employee",
				}, nil, LoginRequestFieldDescriptions)
				mockDB.On("QueryRow", mock.Anything, mock.AnythingOfType("string"), "test@example.com").Return(mockRow)

				errorCmd := redis.NewStatusCmd(context.Background())
				errorCmd.SetErr(errors.New("redis error"))

				mockRedis.On("Set", mock.Anything, mock.MatchedBy(func(key string) bool {
					return strings.Contains(key, "access_token:")
				}), "valid", mock.AnythingOfType("time.Duration")).Return(errorCmd)
			},
			expectError:   true,
			errorContains: "redis error",
		},
		{
			name: "user not found",
			loginReq: &entity.LoginRequest{
				Email:    "notfound@example.com",
				Password: "password123",
			},
			setupMocks: func(mockDB *MockDB, mockRedis *MockRedis) {
				mockRow := NewMockRow(nil, pgx.ErrNoRows, LoginRequestFieldDescriptions)
				mockDB.On("QueryRow", mock.Anything, mock.AnythingOfType("string"), "notfound@example.com").Return(mockRow)
			},
			expectError:   true,
			errorContains: "user with this email not found",
		},
		{
			name: "database error",
			loginReq: &entity.LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			setupMocks: func(mockDB *MockDB, mockRedis *MockRedis) {
				mockRow := NewMockRow(nil, errors.New("database connection error"), LoginRequestFieldDescriptions)
				mockDB.On("QueryRow", mock.Anything, mock.AnythingOfType("string"), "test@example.com").Return(mockRow)
			},
			expectError:   true,
			errorContains: "database connection error",
		},
		{
			name: "invalid password",
			loginReq: &entity.LoginRequest{
				Email:    "test@example.com",
				Password: "wrongpassword",
			},
			setupMocks: func(mockDB *MockDB, mockRedis *MockRedis) {
				hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)
				passwordStr := string(hashedPassword)

				mockRow := NewMockRow([]interface{}{
					uint64(1), "test@example.com", &passwordStr, "employee",
				}, nil, LoginRequestFieldDescriptions)
				mockDB.On("QueryRow", mock.Anything, mock.AnythingOfType("string"), "test@example.com").Return(mockRow)
			},
			expectError: true,
		},
		{
			name: "redis error on access token",
			loginReq: &entity.LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			setupMocks: func(mockDB *MockDB, mockRedis *MockRedis) {
				hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
				passwordStr := string(hashedPassword)

				mockRow := NewMockRow([]interface{}{
					uint64(1), "test@example.com", passwordStr, "employee",
				}, nil, LoginRequestFieldDescriptions)
				mockDB.On("QueryRow", mock.Anything, mock.AnythingOfType("string"), "test@example.com").Return(mockRow)

				errorCmd := redis.NewStatusCmd(context.Background())
				errorCmd.SetErr(errors.New("redis error"))
				mockRedis.On("Set", mock.Anything, mock.MatchedBy(func(key string) bool {
					return strings.Contains(key, "access_token:")
				}), "valid", mock.AnythingOfType("time.Duration")).Return(errorCmd)
			},
			expectError:   true,
			errorContains: "redis error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDB{}
			mockRedis := &MockRedis{}
			deps := CreateTestDependencies(mockDB, mockRedis)

			tt.setupMocks(mockDB, mockRedis)

			controller := NewAuthController(deps)

			accessToken, refreshToken, err := controller.AuthLogin(tt.loginReq)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Empty(t, accessToken)
				assert.Empty(t, refreshToken)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, accessToken)
				assert.NotEmpty(t, refreshToken)
			}

			mockDB.AssertExpectations(t)
			mockRedis.AssertExpectations(t)
		})
	}
}

func TestAuthController_createToken(t *testing.T) {
	tests := []struct {
		name        string
		employee    entity.Employee
		tokenType   string
		expectError bool
	}{
		{
			name:        "create access token",
			employee:    CreateTestEmployee(),
			tokenType:   "access",
			expectError: false,
		},
		{
			name:        "create refresh token",
			employee:    CreateTestEmployee(),
			tokenType:   "refresh",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDB{}
			mockRedis := &MockRedis{}
			deps := CreateTestDependencies(mockDB, mockRedis)

			controller := NewAuthController(deps)
			token, err := controller.createToken(tt.employee, tt.tokenType)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, token)

				parsedToken, err := jwt.ParseWithClaims(token, &entity.Claims{}, func(_ *jwt.Token) (interface{}, error) {
					return []byte(deps.Config.Server.JWTSecret), nil
				})

				assert.NoError(t, err)
				assert.True(t, parsedToken.Valid)

				claims, ok := parsedToken.Claims.(*entity.Claims)
				assert.True(t, ok)
				assert.Equal(t, *tt.employee.ID, claims.ID)
				assert.Equal(t, *tt.employee.Email, claims.Email)
				assert.Equal(t, tt.employee.Role, claims.Role)
				assert.NotEmpty(t, claims.TokenID)
			}
		})
	}
}

func TestGenerateTokenID(t *testing.T) {
	mockDB := &MockDB{}
	mockRedis := &MockRedis{}
	deps := CreateTestDependencies(mockDB, mockRedis)

	tokenID1, err1 := generateTokenID(deps.Logger)
	assert.NoError(t, err1)
	assert.NotEmpty(t, tokenID1)
	assert.Equal(t, TokenSize*2, len(tokenID1))

	tokenID2, err2 := generateTokenID(deps.Logger)
	assert.NoError(t, err2)
	assert.NotEmpty(t, tokenID2)

	assert.NotEqual(t, tokenID1, tokenID2)
}

func TestAuthController_CheckUserToken(t *testing.T) {
	tests := []struct {
		name          string
		authHeader    string
		setupMocks    func(*MockDB, *MockRedis, string)
		expectError   bool
		errorContains string
	}{
		{
			name:       "valid token",
			authHeader: "Bearer token",
			setupMocks: func(mockDB *MockDB, mockRedis *MockRedis, token string) {
				mockRedis.On("Get", mock.Anything, "access_token:"+token).Return(redis.NewStringCmd(context.Background()))
			},
			expectError: false,
		},
		{
			name:          "invalid bearer format",
			authHeader:    "InvalidToken",
			setupMocks:    func(mockDB *MockDB, mockRedis *MockRedis, token string) {},
			expectError:   true,
			errorContains: "invalid bearer token",
		},
		{
			name:       "token revoked",
			authHeader: "Bearer revoked-token",
			setupMocks: func(mockDB *MockDB, mockRedis *MockRedis, token string) {
				revokedCmd := redis.NewStringCmd(context.Background())
				revokedCmd.SetErr(redis.Nil)
				mockRedis.On("Get", mock.Anything, "access_token:"+token).Return(revokedCmd)
			},
			expectError:   true,
			errorContains: "token revoked",
		},
		{
			name:       "invalid token format",
			authHeader: "Bearer invalid-token-format",
			setupMocks: func(mockDB *MockDB, mockRedis *MockRedis, token string) {
				mockRedis.On("Get", mock.Anything, "access_token:invalid-token-format").Return(redis.NewStringCmd(context.Background()))
			},
			expectError:   true,
			errorContains: "invalid token",
		},
		{
			name:       "expired token",
			authHeader: "Bearer expired-token",
			setupMocks: func(mockDB *MockDB, mockRedis *MockRedis, token string) {
				mockRedis.On("Get", mock.Anything, "access_token:"+token).Return(redis.NewStringCmd(context.Background()))
			},
			expectError:   true,
			errorContains: "invalid token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDB{}
			mockRedis := &MockRedis{}
			deps := CreateTestDependencies(mockDB, mockRedis)

			controller := NewAuthController(deps)

			var actualToken string

			switch tt.name {
			case "token revoked":
				claims := &entity.Claims{
					ID:      1,
					Email:   "test@example.com",
					Role:    "employee",
					TokenID: "revoked-token-id",
					RegisteredClaims: jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
						IssuedAt:  jwt.NewNumericDate(time.Now()),
					},
				}

				jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				tokenString, _ := jwtToken.SignedString([]byte("test-secret-key"))
				actualToken = tokenString
				tt.authHeader = "Bearer " + tokenString
			case "expired token":
				claims := &entity.Claims{
					ID:      1,
					Email:   "test@example.com",
					Role:    "employee",
					TokenID: "test-token-id",
					RegisteredClaims: jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
						IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
					},
				}

				jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				tokenString, _ := jwtToken.SignedString([]byte("test-secret-key"))
				actualToken = tokenString
				tt.authHeader = "Bearer " + tokenString
			case "valid token":
				claims := &entity.Claims{
					ID:      1,
					Email:   "test@example.com",
					Role:    "employee",
					TokenID: "test-token-id",
					RegisteredClaims: jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
						IssuedAt:  jwt.NewNumericDate(time.Now().Add(168 * time.Hour)),
					},
				}

				jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				tokenString, _ := jwtToken.SignedString([]byte("test-secret-key"))
				actualToken = tokenString
				tt.authHeader = "Bearer " + tokenString
			}

			if actualToken == "" {
				actualToken = strings.TrimPrefix(tt.authHeader, "Bearer ")
			}

			fmt.Println(tt.authHeader)

			tt.setupMocks(mockDB, mockRedis, actualToken)

			result, err := controller.CheckUserToken(tt.authHeader)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, uint64(1), result.ID)
				assert.Equal(t, "test@example.com", result.Email)
				assert.Equal(t, "employee", result.Role)
			}

			mockRedis.AssertExpectations(t)
		})
	}
}

func TestNewAuthController(t *testing.T) {
	mockDB := &MockDB{}
	mockRedis := &MockRedis{}
	deps := CreateTestDependencies(mockDB, mockRedis)

	controller := NewAuthController(deps)

	assert.NotNil(t, controller)
	assert.Equal(t, deps, controller.deps)
}
