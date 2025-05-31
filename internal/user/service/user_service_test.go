package service

import (
	"context"
	"errors"
	"testing"

	"github.com/ridloal/e-commerce-go-microservices/internal/user/domain"
	"github.com/ridloal/e-commerce-go-microservices/internal/user/repository" // Untuk ErrUserConflict, dll.
	"github.com/ridloal/e-commerce-go-microservices/internal/user/repository/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
	// Anda perlu membuat package mocks atau meletakkannya di tempat yang sesuai
	// Misalnya: "github.com/your_username/your_project/mocks/user_repository_mock"
	// Untuk contoh ini, kita asumsikan ada di package yang sama untuk simplisitas, tapi itu tidak ideal.
	// Sebaiknya buat package terpisah untuk mocks, misal:
	// "github.com/ridloal/e-commerce-go-microservices/internal/user/repository/mocks"
)

func TestUserService_Register(t *testing.T) {
	mockRepo := new(mocks.MockUserRepository)
	userServiceInstance := NewUserService(mockRepo) // Gunakan konstruktor asli

	ctx := context.TODO()
	registerReq := domain.RegisterRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	t.Run("Successful registration", func(t *testing.T) {
		// Setup mock expectation
		// mock.AnythingOfType("*domain.User") karena password hash akan berbeda setiap kali
		mockRepo.On("CreateUser", ctx, mock.AnythingOfType("*domain.User")).Return(nil).Once()

		user, err := userServiceInstance.Register(ctx, registerReq)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, registerReq.Email, user.Email)
		assert.NotEmpty(t, user.ID) // ID di-set oleh mock
		mockRepo.AssertExpectations(t)
	})

	t.Run("User already exists", func(t *testing.T) {
		mockRepo.On("CreateUser", ctx, mock.AnythingOfType("*domain.User")).Return(repository.ErrUserConflict).Once()

		user, err := userServiceInstance.Register(ctx, registerReq)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.EqualError(t, err, ErrUserAlreadyExists.Error()) // Membandingkan dengan error yang didefinisikan di service
		mockRepo.AssertExpectations(t)
	})

	t.Run("Repository error on CreateUser", func(t *testing.T) {
		expectedErr := errors.New("database error")
		mockRepo.On("CreateUser", ctx, mock.AnythingOfType("*domain.User")).Return(expectedErr).Once()

		user, err := userServiceInstance.Register(ctx, registerReq)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "could not save user") // Cek pembungkusan error
		mockRepo.AssertExpectations(t)
	})
}

func TestUserService_Login(t *testing.T) {
	mockRepo := new(mocks.MockUserRepository)
	userServiceInstance := NewUserService(mockRepo)
	ctx := context.TODO()

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	mockUser := &domain.User{
		ID:           "user-123",
		Email:        "test@example.com",
		PasswordHash: string(hashedPassword),
	}

	loginReq := domain.LoginRequest{
		Identifier: "test@example.com",
		Password:   "password123",
	}

	t.Run("Successful login", func(t *testing.T) {
		mockRepo.On("GetUserByIdentifier", ctx, loginReq.Identifier).Return(mockUser, nil).Once()

		resp, err := userServiceInstance.Login(ctx, loginReq)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, mockUser.ID, resp.User.ID)
		assert.NotEmpty(t, resp.Token)
		mockRepo.AssertExpectations(t)
	})

	t.Run("User not found", func(t *testing.T) {
		mockRepo.On("GetUserByIdentifier", ctx, loginReq.Identifier).Return(nil, repository.ErrUserNotFound).Once()

		resp, err := userServiceInstance.Login(ctx, loginReq)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.EqualError(t, err, ErrInvalidCredentials.Error())
		mockRepo.AssertExpectations(t)
	})

	t.Run("Incorrect password", func(t *testing.T) {
		// Password hash di mockUser tidak akan cocok dengan "wrongpassword"
		mockRepo.On("GetUserByIdentifier", ctx, loginReq.Identifier).Return(mockUser, nil).Once()

		reqWithWrongPass := domain.LoginRequest{
			Identifier: "test@example.com",
			Password:   "wrongpassword",
		}
		resp, err := userServiceInstance.Login(ctx, reqWithWrongPass)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.EqualError(t, err, ErrInvalidCredentials.Error())
		mockRepo.AssertExpectations(t)
	})

	t.Run("Repository error on GetUserByIdentifier", func(t *testing.T) {
		expectedRepoErr := errors.New("some db error")
		mockRepo.On("GetUserByIdentifier", ctx, loginReq.Identifier).Return(nil, expectedRepoErr).Once()

		resp, err := userServiceInstance.Login(ctx, loginReq)

		assert.Error(t, err)
		assert.Nil(t, resp)
		// Karena service layer mengembalikan ErrInvalidCredentials untuk semua error dari repo.GetUserByIdentifier selain ErrUserNotFound
		assert.EqualError(t, err, ErrInvalidCredentials.Error())
		mockRepo.AssertExpectations(t)
	})
}
