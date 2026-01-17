package auth

import (
	"time"
)

// User represents a user in the system
type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"` // Never expose password hash in JSON
	CreatedAt    int64     `json:"created_at"`
	UpdatedAt    int64     `json:"updated_at"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents a successful login response
type LoginResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// InitializeUserRequest represents a request to initialize the first admin user
type InitializeUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// NewUser creates a new user with timestamps
func NewUser(username, passwordHash string) *User {
	now := time.Now().Unix()
	return &User{
		Username:     username,
		PasswordHash: passwordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}
