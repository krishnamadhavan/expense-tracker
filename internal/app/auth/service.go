package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/krishnamadhavan/expense-tracker/internal/db"
	"github.com/krishnamadhavan/expense-tracker/internal/domain"
	"github.com/krishnamadhavan/expense-tracker/internal/ports"
	"golang.org/x/crypto/argon2"
)

// Service handles password login, sessions, API tokens, and bootstrap.
type Service struct {
	Users    ports.UserRepository
	Sessions ports.SessionRepository
	Tokens   ports.APITokenRepository

	AbsoluteTTL time.Duration
	IdleTTL     time.Duration
}

// BootstrapIfEmpty creates the first admin on the seed household when users=0 and password is set.
func (s *Service) BootstrapIfEmpty(ctx context.Context, email, password string) error {
	if password == "" {
		return nil
	}
	n, err := s.Users.Count(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	hh, err := uuid.Parse(db.DefaultSeedHouseholdID)
	if err != nil {
		return err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	_, err = s.Users.Create(ctx, hh, email, hash, domain.UserRoleAdmin)
	return err
}

// Login validates password and creates a session; returns raw session token and CSRF secret.
func (s *Service) Login(ctx context.Context, email, password string) (rawToken, csrf string, expires time.Time, err error) {
	u, err := s.Users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return "", "", time.Time{}, errInvalidCredentials
		}
		return "", "", time.Time{}, err
	}
	if !CheckPassword(u.PasswordHash, password) {
		return "", "", time.Time{}, errInvalidCredentials
	}
	rawToken, tokenHash, err := newOpaqueToken()
	if err != nil {
		return "", "", time.Time{}, err
	}
	csrf, err = randomURLString(32)
	if err != nil {
		return "", "", time.Time{}, err
	}
	now := time.Now().UTC()
	abs := s.AbsoluteTTL
	if abs == 0 {
		abs = 12 * time.Hour
	}
	idle := s.IdleTTL
	if idle == 0 {
		idle = 2 * time.Hour
	}
	expires = now.Add(abs)
	rec := ports.SessionRecord{
		ID:            uuid.New(),
		UserID:        u.ID,
		TokenHash:     tokenHash,
		CSRFSecret:    csrf,
		ExpiresAt:     expires,
		IdleExpiresAt: now.Add(idle),
	}
	if err := s.Sessions.Create(ctx, rec); err != nil {
		return "", "", time.Time{}, err
	}
	return rawToken, csrf, expires, nil
}

// Logout deletes the session by id.
func (s *Service) Logout(ctx context.Context, sessionID domain.SessionID) error {
	return s.Sessions.Delete(ctx, sessionID)
}

// ResolveSession loads principal from session cookie value (raw token).
func (s *Service) ResolveSession(ctx context.Context, rawToken string) (ports.Principal, error) {
	if rawToken == "" {
		return ports.Principal{}, domain.ErrNotFound
	}
	hash := hashToken(rawToken)
	sess, err := s.Sessions.GetByTokenHash(ctx, hash)
	if err != nil {
		return ports.Principal{}, err
	}
	now := time.Now().UTC()
	if now.After(sess.ExpiresAt) || now.After(sess.IdleExpiresAt) {
		_ = s.Sessions.Delete(ctx, sess.ID)
		return ports.Principal{}, domain.ErrNotFound
	}
	idle := s.IdleTTL
	if idle == 0 {
		idle = 2 * time.Hour
	}
	_ = s.Sessions.TouchIdle(ctx, sess.ID, now.Add(idle))
	u, err := s.Users.GetByID(ctx, sess.UserID)
	if err != nil {
		return ports.Principal{}, err
	}
	return ports.Principal{
		UserID:      u.ID,
		HouseholdID: u.HouseholdID,
		Email:       u.Email,
		SessionID:   sess.ID,
		CSRFSecret:  sess.CSRFSecret,
		ViaBearer:   false,
		Scope:       domain.TokenScopeWrite,
	}, nil
}

// ResolveBearer loads principal from Authorization bearer secret.
func (s *Service) ResolveBearer(ctx context.Context, rawToken string) (ports.Principal, error) {
	if rawToken == "" {
		return ports.Principal{}, domain.ErrNotFound
	}
	hash := hashToken(rawToken)
	tok, err := s.Tokens.GetByTokenHash(ctx, hash)
	if err != nil {
		return ports.Principal{}, err
	}
	if tok.RevokedAt != nil || time.Now().UTC().After(tok.ExpiresAt) {
		return ports.Principal{}, domain.ErrNotFound
	}
	u, err := s.Users.GetByID(ctx, tok.UserID)
	if err != nil {
		return ports.Principal{}, err
	}
	return ports.Principal{
		UserID:      u.ID,
		HouseholdID: u.HouseholdID,
		Email:       u.Email,
		ViaBearer:   true,
		Scope:       tok.Scopes,
	}, nil
}

// CreateAPIToken issues a new bearer token; returns the one-time secret.
func (s *Service) CreateAPIToken(ctx context.Context, userID domain.UserID, name string, scope domain.TokenScope, ttl time.Duration) (raw string, rec ports.APITokenRecord, err error) {
	if scope == "" {
		scope = domain.TokenScopeWrite
	}
	if !scope.Valid() {
		return "", ports.APITokenRecord{}, fmt.Errorf("%w: scopes", domain.ErrInvalidArgument)
	}
	if ttl == 0 {
		ttl = 90 * 24 * time.Hour
	}
	raw, hash, err := newOpaqueToken()
	if err != nil {
		return "", ports.APITokenRecord{}, err
	}
	prefix := "et_live_"
	if len(raw) > 8 {
		prefix = "et_live_" + raw[:8]
	}
	rec = ports.APITokenRecord{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      name,
		Prefix:    prefix,
		TokenHash: hash,
		Scopes:    scope,
		ExpiresAt: time.Now().UTC().Add(ttl),
	}
	if err := s.Tokens.Create(ctx, rec); err != nil {
		return "", ports.APITokenRecord{}, err
	}
	// Client sees full secret once: prefix-style et_live_<base64>
	display := "et_live_" + raw
	return display, rec, nil
}

var errInvalidCredentials = errors.New("invalid credentials")

// ErrInvalidCredentials is returned on bad login.
func ErrInvalidCredentials() error { return errInvalidCredentials }

func IsInvalidCredentials(err error) bool {
	return errors.Is(err, errInvalidCredentials)
}

// HashPassword uses argon2id.
func HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	// time=1, memory=64MB, threads=4, keyLen=32
	key := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	// encode: argon2id$saltHex$keyHex
	return "argon2id$" + hex.EncodeToString(salt) + "$" + hex.EncodeToString(key), nil
}

// CheckPassword verifies an argon2id encoded hash.
func CheckPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 3 || parts[0] != "argon2id" {
		return false
	}
	salt, err := hex.DecodeString(parts[1])
	if err != nil {
		return false
	}
	want, err := hex.DecodeString(parts[2])
	if err != nil {
		return false
	}
	got := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	if len(got) != len(want) {
		return false
	}
	var v byte
	for i := range got {
		v |= got[i] ^ want[i]
	}
	return v == 0
}

func newOpaqueToken() (raw, hash string, err error) {
	raw, err = randomURLString(32)
	if err != nil {
		return "", "", err
	}
	return raw, hashToken(raw), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func randomURLString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
