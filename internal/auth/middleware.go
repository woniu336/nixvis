package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	// ContextKeyUserID is the key used to store user ID in gin context
	ContextKeyUserID = "user_id"
	// ContextKeyUsername is the key used to store username in gin context
	ContextKeyUsername = "username"
)

// AuthMiddleware creates a middleware that requires authentication
func (m *Manager) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		// Check for token in Authorization header
		if authHeader == "" {
			// Check for token in cookie
			if token, err := c.Cookie("auth_token"); err == nil && token != "" {
				if claims, err := m.ValidateToken(token); err == nil {
					c.Set(ContextKeyUserID, claims.UserID)
					c.Set(ContextKeyUsername, claims.Username)
					c.Next()
					return
				}
			}
			// For API requests, return JSON error
			if strings.HasPrefix(c.Request.URL.Path, "/api/") {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			} else {
				// For page requests, redirect to login
				c.Redirect(http.StatusFound, "/login")
			}
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>" format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			// For API requests, return JSON error
			if strings.HasPrefix(c.Request.URL.Path, "/api/") {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
			} else {
				// For page requests, redirect to login
				c.Redirect(http.StatusFound, "/login")
			}
			c.Abort()
			return
		}

		tokenString := parts[1]
		claims, err := m.ValidateToken(tokenString)
		if err != nil {
			// For API requests, return JSON error
			if strings.HasPrefix(c.Request.URL.Path, "/api/") {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			} else {
				// For page requests, redirect to login
				c.Redirect(http.StatusFound, "/login")
			}
			c.Abort()
			return
		}

		// Store user info in context
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyUsername, claims.Username)

		c.Next()
	}
}

// OptionalAuthMiddleware creates a middleware that optionally authenticates
// It doesn't reject unauthenticated requests, but sets user info if available
func (m *Manager) OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				if claims, err := m.ValidateToken(parts[1]); err == nil {
					c.Set(ContextKeyUserID, claims.UserID)
					c.Set(ContextKeyUsername, claims.Username)
				}
			}
		} else {
			// Check cookie
			if token, err := c.Cookie("auth_token"); err == nil {
				if claims, err := m.ValidateToken(token); err == nil {
					c.Set(ContextKeyUserID, claims.UserID)
					c.Set(ContextKeyUsername, claims.Username)
				}
			}
		}

		c.Next()
	}
}

// GetUserID retrieves the user ID from the gin context
func GetUserID(c *gin.Context) (int64, bool) {
	userID, exists := c.Get(ContextKeyUserID)
	if !exists {
		return 0, false
	}
	return userID.(int64), true
}

// GetUsername retrieves the username from the gin context
func GetUsername(c *gin.Context) (string, bool) {
	username, exists := c.Get(ContextKeyUsername)
	if !exists {
		return "", false
	}
	return username.(string), true
}
