package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/krishnamadhavan/expense-tracker/internal/domain"
)

// UserRecord is the persisted user row needed for auth.
type UserRecord struct {
	ID           domain.UserID
	HouseholdID  domain.HouseholdID
	Email        string
	PasswordHash string
	Role         domain.UserRole
}

// SessionRecord is a server-side web session.
type SessionRecord struct {
	ID            domain.SessionID
	UserID        domain.UserID
	TokenHash     string
	CSRFSecret    string
	ExpiresAt     time.Time
	IdleExpiresAt time.Time
}

// APITokenRecord is a bearer token (hash stored).
type APITokenRecord struct {
	ID         domain.APITokenID
	UserID     domain.UserID
	Name       string
	Prefix     string
	TokenHash  string
	Scopes     domain.TokenScope
	ExpiresAt  time.Time
	RevokedAt  *time.Time
}

// UserRepository supports login and bootstrap.
type UserRepository interface {
	Count(ctx context.Context) (int, error)
	Create(ctx context.Context, householdID domain.HouseholdID, email, passwordHash string, role domain.UserRole) (UserRecord, error)
	GetByEmail(ctx context.Context, email string) (UserRecord, error)
	GetByID(ctx context.Context, id domain.UserID) (UserRecord, error)
}

// SessionRepository stores opaque session tokens (hashed).
type SessionRepository interface {
	Create(ctx context.Context, s SessionRecord) error
	GetByTokenHash(ctx context.Context, tokenHash string) (SessionRecord, error)
	TouchIdle(ctx context.Context, id domain.SessionID, idleExpiresAt time.Time) error
	Delete(ctx context.Context, id domain.SessionID) error
}

// APITokenRepository stores API bearer tokens.
type APITokenRepository interface {
	Create(ctx context.Context, t APITokenRecord) error
	GetByTokenHash(ctx context.Context, tokenHash string) (APITokenRecord, error)
	ListByUser(ctx context.Context, userID domain.UserID) ([]APITokenRecord, error)
	Revoke(ctx context.Context, userID domain.UserID, id domain.APITokenID) error
}

// Principal is the authenticated caller for a request.
type Principal struct {
	UserID      domain.UserID
	HouseholdID domain.HouseholdID
	Email       string
	// SessionID set for cookie auth; zero for bearer.
	SessionID   uuid.UUID
	CSRFSecret  string
	// ViaBearer is true when Authorization: Bearer was used (CSRF exempt).
	ViaBearer   bool
	// Scope for bearer tokens; write implies read. Session auth has write.
	Scope       domain.TokenScope
}
