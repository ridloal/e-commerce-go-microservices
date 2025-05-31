package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ridloal/e-commerce-go-microservices/internal/platform/logger"
	"github.com/ridloal/e-commerce-go-microservices/internal/user/domain"
	"github.com/ridloal/e-commerce-go-microservices/internal/user/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid email/phone or password")
	ErrUserAlreadyExists  = errors.New("user with this email or phone number already exists")
)

var jwtSecretKey []byte

func init() {
	secret := os.Getenv("JWT_SECRET_KEY")
	if secret == "" {
		log.Println("Warning: JWT_SECRET_KEY not set, using default insecure key")
		secret = "your-very-secret-key-for-jwt" // fallback
	}
	jwtSecretKey = []byte(secret)
}

type UserService interface {
	Register(ctx context.Context, req domain.RegisterRequest) (*domain.User, error)
	Login(ctx context.Context, req domain.LoginRequest) (*domain.LoginResponse, error)
}

type userService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) UserService {
	return &userService{repo: repo}
}

func (s *userService) Register(ctx context.Context, req domain.RegisterRequest) (*domain.User, error) {
	// Validasi dasar (sebagian sudah dilakukan oleh Gin `binding:"required"`)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.PhoneNumber != nil {
		*req.PhoneNumber = strings.TrimSpace(*req.PhoneNumber)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("Register: failed to hash password", err)
		return nil, fmt.Errorf("could not process registration: %w", err)
	}

	user := &domain.User{
		Email:        req.Email,
		PhoneNumber:  req.PhoneNumber,
		PasswordHash: string(hashedPassword),
	}

	err = s.repo.CreateUser(ctx, user)
	if err != nil {
		if errors.Is(err, repository.ErrUserConflict) {
			return nil, ErrUserAlreadyExists
		}
		logger.Error("Register: failed to create user in repo", err)
		return nil, fmt.Errorf("could not save user: %w", err)
	}

	user.PasswordHash = "" // Hapus sebelum dikembalikan
	return user, nil
}

func (s *userService) Login(ctx context.Context, req domain.LoginRequest) (*domain.LoginResponse, error) {
	req.Identifier = strings.TrimSpace(req.Identifier)

	user, err := s.repo.GetUserByIdentifier(ctx, req.Identifier)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		logger.Error("Login: failed to get user by identifier", err)
		return nil, ErrInvalidCredentials
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil { // Password tidak cocok
		return nil, ErrInvalidCredentials
	}

	// Buat JWT Token
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"exp":     time.Now().Add(time.Hour * 72).Unix(), // Token berlaku 72 jam
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecretKey)
	if err != nil {
		logger.Error("Login: failed to sign token", err)
		return nil, fmt.Errorf("could not generate token: %w", err)
	}

	user.PasswordHash = "" // Hapus sebelum dikembalikan
	return &domain.LoginResponse{
		User:  *user,
		Token: tokenString,
	}, nil
}
