package auth

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/outpost/outpost/internal/database"
)

const (
	SessionDuration = 7 * 24 * time.Hour // 7 days
	TokenLength     = 32
)

type Service struct {
	db *database.Database
}

func New(db *database.Database) *Service {
	return &Service{db: db}
}

// HashPassword creates a bcrypt hash of the password
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword compares a password with a hash
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateToken creates a random session token
func GenerateToken() (string, error) {
	bytes := make([]byte, TokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// Login authenticates a user and creates a session
func (s *Service) Login(username, password string) (*database.Session, *database.User, error) {
	user, err := s.db.GetUserByUsername(username)
	if err != nil {
		return nil, nil, err
	}

	if !CheckPassword(password, user.PasswordHash) {
		return nil, nil, bcrypt.ErrMismatchedHashAndPassword
	}

	token, err := GenerateToken()
	if err != nil {
		return nil, nil, err
	}

	session := &database.Session{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(SessionDuration),
	}

	if err := s.db.CreateSession(session); err != nil {
		return nil, nil, err
	}

	return session, user, nil
}

// Logout invalidates a session
func (s *Service) Logout(token string) error {
	return s.db.DeleteSession(token)
}

// ValidateSession checks if a session token is valid and returns the user
func (s *Service) ValidateSession(token string) (*database.User, error) {
	session, err := s.db.GetSessionByToken(token)
	if err != nil {
		return nil, err
	}

	if time.Now().After(session.ExpiresAt) {
		s.db.DeleteSession(token)
		return nil, bcrypt.ErrMismatchedHashAndPassword // Session expired
	}

	return s.db.GetUserByID(session.UserID)
}

// CreateUser creates a new user with hashed password
func (s *Service) CreateUser(username, password, role string) (*database.User, error) {
	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &database.User{
		Username:     username,
		PasswordHash: hash,
		Role:         role,
	}

	if err := s.db.CreateUser(user); err != nil {
		return nil, err
	}

	return user, nil
}

// EnsureAdminExists creates a default admin if no users exist
func (s *Service) EnsureAdminExists() error {
	count, err := s.db.CountUsers()
	if err != nil {
		return err
	}

	if count == 0 {
		_, err = s.CreateUser("admin", "admin", "admin")
		if err != nil {
			return err
		}
	}

	return nil
}

// CleanupExpiredSessions removes expired sessions
func (s *Service) CleanupExpiredSessions() error {
	return s.db.DeleteExpiredSessions()
}
