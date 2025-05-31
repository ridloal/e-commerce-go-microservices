package mocks

import (
	"context"
	"time"

	"github.com/ridloal/e-commerce-go-microservices/internal/user/domain"
	"github.com/stretchr/testify/mock"
)

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) CreateUser(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	// Jika service Anda bergantung pada ID yang diset oleh repo, mock ini perlu mengisinya.
	// Contoh sederhana:
	if user != nil && args.Error(0) == nil { // Jika tidak ada error
		user.ID = "mocked-user-id"
		user.CreatedAt = time.Now()
		user.UpdatedAt = time.Now()
	}
	return args.Error(0)
}

func (m *MockUserRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if u := args.Get(0); u != nil {
		return u.(*domain.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserRepository) GetUserByPhoneNumber(ctx context.Context, phoneNumber string) (*domain.User, error) {
	args := m.Called(ctx, phoneNumber)
	if u := args.Get(0); u != nil {
		return u.(*domain.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserRepository) GetUserByIdentifier(ctx context.Context, identifier string) (*domain.User, error) {
	args := m.Called(ctx, identifier)
	if u := args.Get(0); u != nil {
		return u.(*domain.User), args.Error(1)
	}
	return nil, args.Error(1)
}
