package auth

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// SQLiteUserStore implements UserStore for SQLite database
type SQLiteUserStore struct {
	db *sql.DB
}

// NewSQLiteUserStore creates a new SQLite user store
func NewSQLiteUserStore(db *sql.DB) *SQLiteUserStore {
	return &SQLiteUserStore{db: db}
}

// InitSchema creates the users table if it doesn't exist
func (s *SQLiteUserStore) InitSchema() error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);`

	_, err := s.db.Exec(query)
	return err
}

// GetUserByUsername retrieves a user by username
func (s *SQLiteUserStore) GetUserByUsername(username string) (*User, error) {
	query := `SELECT id, username, password_hash, created_at, updated_at FROM users WHERE username = ?`

	var user User
	err := s.db.QueryRow(query, username).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUserByID retrieves a user by ID
func (s *SQLiteUserStore) GetUserByID(id int64) (*User, error) {
	query := `SELECT id, username, password_hash, created_at, updated_at FROM users WHERE id = ?`

	var user User
	err := s.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// CreateUser creates a new user
func (s *SQLiteUserStore) CreateUser(user *User) error {
	query := `
	INSERT INTO users (username, password_hash, created_at, updated_at)
	VALUES (?, ?, ?, ?)`

	result, err := s.db.Exec(query, user.Username, user.PasswordHash, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		// Check for unique constraint violation
		if err.Error() == "UNIQUE constraint failed: users.username" {
			return ErrUserAlreadyExists
		}
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	user.ID = id
	return nil
}

// UpdatePassword updates a user's password
func (s *SQLiteUserStore) UpdatePassword(userID int64, passwordHash string) error {
	query := `UPDATE users SET password_hash = ?, updated_at = strftime('%s', 'now') WHERE id = ?`

	result, err := s.db.Exec(query, passwordHash, userID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}

// UserExists checks if any user exists in the database
func (s *SQLiteUserStore) UserExists() (bool, error) {
	query := `SELECT COUNT(*) FROM users LIMIT 1`

	var count int
	err := s.db.QueryRow(query).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetAllUsers retrieves all users (for admin purposes)
func (s *SQLiteUserStore) GetAllUsers() ([]*User, error) {
	query := `SELECT id, username, created_at, updated_at FROM users ORDER BY id`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.ID, &user.Username, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, &user)
	}

	return users, nil
}

// DeleteUser deletes a user by ID
func (s *SQLiteUserStore) DeleteUser(userID int64) error {
	query := `DELETE FROM users WHERE id = ?`

	result, err := s.db.Exec(query, userID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}

// UpdateUsername updates a user's username
func (s *SQLiteUserStore) UpdateUsername(userID int64, newUsername string) error {
	query := `UPDATE users SET username = ?, updated_at = strftime('%s', 'now') WHERE id = ?`

	result, err := s.db.Exec(query, newUsername, userID)
	if err != nil {
		// Check for unique constraint violation
		if err.Error() == fmt.Sprintf("UNIQUE constraint failed: users.username") {
			return ErrUserAlreadyExists
		}
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}
