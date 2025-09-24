package controllers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/adamanr/employes_service/internal/entity"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

const TokenSize = 16

type AuthController struct {
	deps *Dependens
}

func NewAuthController(deps *Dependens) *AuthController {
	return &AuthController{
		deps: deps,
	}
}

func (c *AuthController) AuthLogin(req *entity.LoginRequest) (string, string, error) {
	var id uint64
	var email, password, role string

	if err := c.deps.DB.QueryRow(context.Background(), "SELECT id, email, password, role FROM employees WHERE email = $1", req.Email).Scan(&id, &email, &password, &role); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.deps.Logger.Warn("user with this email not found", slog.String("email", req.Email))
			return "", "", errors.New("user with this email not found")
		}

		c.deps.Logger.Error("Error querying employee", slog.String("error", err.Error()))
		return "", "", err
	}

	emp := entity.Employee{
		ID:       &id,
		Email:    &email,
		Password: &password,
		Role:     role,
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*emp.Password), []byte(req.Password)); err != nil {
		c.deps.Logger.Warn("Invalid password", slog.String("email", req.Email))
		return "", "", err
	}

	accessToken, err := c.createToken(emp, "access")
	if err != nil {
		return "", "", err
	}

	refreshToken, err := c.createToken(emp, "refresh")
	if err != nil {
		return "", "", err
	}

	ctx := context.Background()
	if err = c.deps.Redis.Set(ctx, "access_token:"+accessToken, "valid", c.deps.Config.Redis.AccessTokenTTL).Err(); err != nil {
		c.deps.Logger.Error("Error setting access token", slog.String("error", err.Error()))
		return "", "", err
	}

	if err = c.deps.Redis.Set(ctx, "refresh_token:"+refreshToken, "valid", c.deps.Config.Redis.RefreshTokenTTL).Err(); err != nil {
		c.deps.Logger.Error("Error setting refresh token", slog.String("error", err.Error()))
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (c *AuthController) createToken(emp entity.Employee, tokenType string) (string, error) {
	tokenID, err := generateTokenID(c.deps.Logger)
	if err != nil {
		c.deps.Logger.Error("Error generating refresh token ID", slog.String("error", err.Error()))
		return "", err
	}

	expiresAt := c.deps.Config.Redis.AccessTokenTTL
	if tokenType == "refresh" {
		expiresAt = c.deps.Config.Redis.RefreshTokenTTL
	}

	claims := entity.Claims{
		ID:      *emp.ID,
		Email:   *emp.Email,
		Role:    emp.Role,
		TokenID: tokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresAt)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(c.deps.Config.Server.JWTSecret))
	if err != nil {
		c.deps.Logger.Error("Error signing token", slog.String("error", err.Error()))
		return "", err
	}

	return tokenStr, nil
}

func generateTokenID(logger *slog.Logger) (string, error) {
	b := make([]byte, TokenSize)
	if _, err := rand.Read(b); err != nil {
		logger.Error("Error generating token ID", slog.String("error", err.Error()))
		return "", err
	}

	return hex.EncodeToString(b), nil
}

func (c *AuthController) CheckUserToken(authHeader string) (*entity.Claims, error) {
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenStr == authHeader {
		c.deps.Logger.Error("Invalid bearer token", slog.String("token", tokenStr))
		return nil, errors.New("invalid bearer token")
	}

	ctx := context.Background()
	if err := c.deps.Redis.Get(ctx, "access_token:"+tokenStr).Err(); errors.Is(err, redis.Nil) {
		c.deps.Logger.Warn("Token revoked", slog.String("token", tokenStr))
		return nil, errors.New("token revoked")
	}

	token, err := jwt.ParseWithClaims(tokenStr, &entity.Claims{}, func(_ *jwt.Token) (any, error) {
		return []byte(c.deps.Config.Server.JWTSecret), nil
	})
	if err != nil {
		c.deps.Logger.Error("Error parsing token", slog.String("error", err.Error()))
		return nil, errors.New("invalid token")
	}

	if claims, ok := token.Claims.(*entity.Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
