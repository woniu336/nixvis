package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrInvalidToken      = errors.New("invalid token")
	ErrTokenExpired      = errors.New("token expired")
)

// JWT claims structure
type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// Manager handles authentication operations
type Manager struct {
	secretKey     string
	userStore     UserStore
	tokenExpiry   time.Duration
	sessionTokens map[string]*Claims // In-memory session storage
	sessionMutex  sync.RWMutex
}

// UserStore defines the interface for user storage operations
type UserStore interface {
	GetUserByUsername(username string) (*User, error)
	CreateUser(user *User) error
	UpdatePassword(userID int64, passwordHash string) error
	UserExists() (bool, error)
}

// NewManager creates a new authentication manager
func NewManager(secretKey string, store UserStore) *Manager {
	return &Manager{
		secretKey:     secretKey,
		userStore:     store,
		tokenExpiry:   24 * time.Hour,
		sessionTokens: make(map[string]*Claims),
	}
}

// generateSecretKey generates a random secret key if none provided
func GenerateSecretKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// HashPassword creates a bcrypt hash of the password
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword checks if the provided password matches the hash
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// Login authenticates a user and returns a JWT token
func (m *Manager) Login(username, password string) (*LoginResponse, error) {
	user, err := m.userStore.GetUserByUsername(username)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !CheckPassword(password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	token, err := m.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		Token:    token,
		Username: user.Username,
	}, nil
}

// InitializeUser creates the first admin user
func (m *Manager) InitializeUser(username, password string) error {
	exists, err := m.userStore.UserExists()
	if err != nil {
		return err
	}
	if exists {
		return ErrUserAlreadyExists
	}

	passwordHash, err := HashPassword(password)
	if err != nil {
		return err
	}

	user := NewUser(username, passwordHash)
	return m.userStore.CreateUser(user)
}

// ChangePassword changes a user's password
func (m *Manager) ChangePassword(userID int64, oldPassword, newPassword string) error {
	user, err := m.userStore.GetUserByUsername("") // We need to get user by ID
	if err != nil {
		return err
	}

	// Verify old password
	if !CheckPassword(oldPassword, user.PasswordHash) {
		return ErrInvalidCredentials
	}

	// Hash new password
	newHash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	return m.userStore.UpdatePassword(userID, newHash)
}

// generateToken generates a JWT token for a user
func (m *Manager) generateToken(user *User) (string, error) {
	expirationTime := time.Now().Add(m.tokenExpiry)

	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   fmt.Sprintf("%d", user.ID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(m.secretKey))
	if err != nil {
		return "", err
	}

	// Store in session
	m.sessionMutex.Lock()
	m.sessionTokens[tokenString] = claims
	m.sessionMutex.Unlock()

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the claims
func (m *Manager) ValidateToken(tokenString string) (*Claims, error) {
	// Check in-memory session first
	m.sessionMutex.RLock()
	if claims, exists := m.sessionTokens[tokenString]; exists {
		m.sessionMutex.RUnlock()
		// Check expiration
		if time.Until(claims.ExpiresAt.Time) > 0 {
			return claims, nil
		}
		m.sessionMutex.RUnlock()
		m.sessionMutex.Lock()
		delete(m.sessionTokens, tokenString)
		m.sessionMutex.Unlock()
		return nil, ErrTokenExpired
	}
	m.sessionMutex.RUnlock()

	// Parse and validate token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.secretKey), nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

// Logout invalidates a token
func (m *Manager) Logout(tokenString string) {
	m.sessionMutex.Lock()
	delete(m.sessionTokens, tokenString)
	m.sessionMutex.Unlock()
}

// IsInitialized checks if the system has been initialized with a user
func (m *Manager) IsInitialized() (bool, error) {
	return m.userStore.UserExists()
}

// GenerateSessionID generates a session ID for password reset or other purposes
func GenerateSessionID() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	hash := sha256.Sum256(bytes)
	return base64.URLEncoding.EncodeToString(hash[:])
}
