package services

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"buffetflow/internal/models"
	"buffetflow/internal/repositories"
)

const (
	passwordIterations = 120000
	passwordSaltSize   = 16
	passwordKeySize    = 32
)

type AuthService struct {
	store *repositories.Store
}

func NewAuthService(store *repositories.Store) *AuthService {
	return &AuthService{store: store}
}

func HashPassword(password string) (string, error) {
	if len(password) < 8 {
		return "", fmt.Errorf("password must have at least eight characters")
	}
	salt := make([]byte, passwordSaltSize)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	key := pbkdf2SHA256([]byte(password), salt, passwordIterations, passwordKeySize)
	return fmt.Sprintf("pbkdf2_sha256$%d$%s$%s", passwordIterations,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)), nil
}

func VerifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2_sha256" {
		return false
	}
	iterations, err := strconv.Atoi(parts[1])
	if err != nil || iterations < 10000 || iterations > 1000000 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}
	actual := pbkdf2SHA256([]byte(password), salt, iterations, len(expected))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func pbkdf2SHA256(password, salt []byte, iterations, keyLength int) []byte {
	result := make([]byte, 0, keyLength)
	for block := 1; len(result) < keyLength; block++ {
		mac := hmac.New(sha256.New, password)
		mac.Write(salt)
		mac.Write([]byte{byte(block >> 24), byte(block >> 16), byte(block >> 8), byte(block)})
		u := mac.Sum(nil)
		t := append([]byte(nil), u...)
		for i := 1; i < iterations; i++ {
			mac = hmac.New(sha256.New, password)
			mac.Write(u)
			u = mac.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		result = append(result, t...)
	}
	return result[:keyLength]
}

func (s *AuthService) EnsureDemoAdmin(ctx context.Context) error {
	hash, err := HashPassword("admin123")
	if err != nil {
		return err
	}
	return s.store.EnsureDemoAdmin(ctx, hash)
}

func (s *AuthService) Login(ctx context.Context, email, password string) (models.User, string, time.Time, error) {
	user, err := s.store.UserByEmail(ctx, strings.TrimSpace(email))
	if err != nil || !user.Active || !VerifyPassword(user.Password, password) {
		return models.User{}, "", time.Time{}, fmt.Errorf("invalid credentials")
	}
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return models.User{}, "", time.Time{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	expires := time.Now().Add(12 * time.Hour)
	if err := s.store.CreateSession(ctx, hashToken(token), user.ID, expires); err != nil {
		return models.User{}, "", time.Time{}, err
	}
	return user, token, expires, nil
}

func (s *AuthService) Authenticate(ctx context.Context, token string) (models.User, error) {
	if token == "" {
		return models.User{}, fmt.Errorf("missing session")
	}
	return s.store.UserBySession(ctx, hashToken(token))
}

func (s *AuthService) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.store.DeleteSession(ctx, hashToken(token))
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
