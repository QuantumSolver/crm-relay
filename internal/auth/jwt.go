package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/yourusername/crm-relay/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// JWTService handles JWT token generation and validation
type JWTService struct {
	secret     string
	expiration time.Duration
}

// NewJWTService creates a new JWT service
func NewJWTService(secret string, expiration int) *JWTService {
	return &JWTService{
		secret:     secret,
		expiration: time.Duration(expiration) * time.Second,
	}
}

// GenerateToken generates a JWT token for a user
func (j *JWTService) GenerateToken(user *models.User) (string, int64, error) {
	if j.secret == "" {
		return "", 0, errors.New("JWT secret is not configured")
	}

	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"role":     user.Role,
		"exp":      time.Now().Add(j.expiration).Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(j.secret))
	if err != nil {
		return "", 0, fmt.Errorf("failed to generate token: %w", err)
	}

	return tokenString, time.Now().Add(j.expiration).Unix(), nil
}

// ValidateToken validates a JWT token and returns the claims
func (j *JWTService) ValidateToken(tokenString string) (*models.JWTClaims, error) {
	if j.secret == "" {
		return nil, errors.New("JWT secret is not configured")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(j.secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	jwtClaims := &models.JWTClaims{
		UserID:   claims["user_id"].(string),
		Username: claims["username"].(string),
		Role:     claims["role"].(string),
	}

	return jwtClaims, nil
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(bytes), nil
}

// VerifyPassword verifies a password against a hash
func VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateAPIKey generates a random API key
func GenerateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate API key: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// GenerateID generates a random ID
func GenerateID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate ID: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}
