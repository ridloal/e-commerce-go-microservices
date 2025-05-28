package domain

import (
	"time"
)

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PhoneNumber  *string   `json:"phone_number,omitempty"` // Pointer agar bisa null
	PasswordHash string    `json:"-"`                      // Jangan kirim password hash ke client
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Untuk registrasi, password plain text
type RegisterRequest struct {
	Email       string  `json:"email" binding:"required,email"`
	PhoneNumber *string `json:"phone_number"`
	Password    string  `json:"password" binding:"required,min=8"`
}

// Untuk login
type LoginRequest struct {
	Identifier string `json:"identifier" binding:"required"` // Bisa email atau phone
	Password   string `json:"password" binding:"required"`
}

type LoginResponse struct {
	User  User   `json:"user"`
	Token string `json:"token"`
}
