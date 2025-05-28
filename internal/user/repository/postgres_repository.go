package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/lib/pq"
	"github.com/ridloal/e-commerce-go-microservices/internal/platform/logger"
	"github.com/ridloal/e-commerce-go-microservices/internal/user/domain"
)

var ErrUserNotFound = errors.New("user not found")
var ErrUserConflict = errors.New("user with this email or phone number already exists")

type UserRepository interface {
	CreateUser(ctx context.Context, user *domain.User) error
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	GetUserByPhoneNumber(ctx context.Context, phoneNumber string) (*domain.User, error)
	GetUserByIdentifier(ctx context.Context, identifier string) (*domain.User, error)
}

type postgresUserRepository struct {
	db *sql.DB
}

func NewPostgresUserRepository(db *sql.DB) UserRepository {
	return &postgresUserRepository{db: db}
}

func (r *postgresUserRepository) CreateUser(ctx context.Context, user *domain.User) error {
	query := `INSERT INTO users (email, phone_number, password_hash, created_at, updated_at)
              VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at, updated_at`

	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	var phoneNumber sql.NullString
	if user.PhoneNumber != nil {
		phoneNumber = sql.NullString{String: *user.PhoneNumber, Valid: true}
	}

	err := r.db.QueryRowContext(ctx, query, user.Email, phoneNumber, user.PasswordHash, user.CreatedAt, user.UpdatedAt).
		Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		// Cek error spesifik PostgreSQL untuk duplikasi (unique violation)
		// Kode error '23505' adalah unique_violation
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			logger.Error("CreateUser: unique violation", err)
			return ErrUserConflict
		}
		logger.Error("CreateUser: failed to insert user", err)
		return err
	}
	return nil
}

func (r *postgresUserRepository) getUserBy(ctx context.Context, field, value string) (*domain.User, error) {
	query := `SELECT id, email, phone_number, password_hash, created_at, updated_at FROM users WHERE ` + field + ` = $1`
	user := &domain.User{}
	var phoneNumber sql.NullString

	err := r.db.QueryRowContext(ctx, query, value).Scan(
		&user.ID, &user.Email, &phoneNumber, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		logger.Error("GetUserBy"+field+": query failed", err)
		return nil, err
	}
	if phoneNumber.Valid {
		user.PhoneNumber = &phoneNumber.String
	}
	return user, nil
}

func (r *postgresUserRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.getUserBy(ctx, "email", email)
}

func (r *postgresUserRepository) GetUserByPhoneNumber(ctx context.Context, phoneNumber string) (*domain.User, error) {
	return r.getUserBy(ctx, "phone_number", phoneNumber)
}

// GetUserByIdentifier bisa mencari berdasarkan email atau nomor telepon.
// Untuk implementasi sederhana, kita bisa coba query email dulu, lalu phone jika tidak ketemu.
// Atau, asumsikan format identifier jelas (misal ada @ untuk email).
func (r *postgresUserRepository) GetUserByIdentifier(ctx context.Context, identifier string) (*domain.User, error) {
	user, err := r.GetUserByEmail(ctx, identifier)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, ErrUserNotFound) { // Error lain selain not found
		return nil, err
	}
	// Coba cari berdasarkan nomor telepon jika bukan email & tidak ditemukan
	return r.GetUserByPhoneNumber(ctx, identifier)
}
