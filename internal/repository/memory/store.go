package memory

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/rhyzoma/locky/internal/domain"
)

type Store struct {
	mu                 sync.RWMutex
	users              map[string]domain.User
	credentials        map[string]domain.PasswordCredential
	sessions           map[string]domain.Session
	verificationTokens map[string]domain.VerificationToken
	events             []domain.OutboxEvent
}

func NewStore() *Store {
	return &Store{
		users:              make(map[string]domain.User),
		credentials:        make(map[string]domain.PasswordCredential),
		sessions:           make(map[string]domain.Session),
		verificationTokens: make(map[string]domain.VerificationToken),
	}
}

type UserStore struct{ store *Store }

func (s *Store) Users() *UserStore { return &UserStore{store: s} }

func (s *Store) Create(ctx context.Context, user domain.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.users {
		if existing.Email == user.Email {
			return domain.ErrEmailExists
		}
	}
	s.users[user.ID] = user
	return nil
}

func (s *Store) ByEmail(ctx context.Context, email string) (domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, user := range s.users {
		if user.Email == email {
			return user, nil
		}
	}
	return domain.User{}, domain.ErrUserNotFound
}

func (s *Store) ByID(ctx context.Context, id string) (domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.users[id]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	return user, nil
}

func (s *Store) Update(ctx context.Context, user domain.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[user.ID]; !ok {
		return domain.ErrUserNotFound
	}
	s.users[user.ID] = user
	return nil
}

func (s *UserStore) Create(ctx context.Context, user domain.User) error {
	return s.store.Create(ctx, user)
}

func (s *UserStore) ByEmail(ctx context.Context, email string) (domain.User, error) {
	return s.store.ByEmail(ctx, email)
}

func (s *UserStore) ByID(ctx context.Context, id string) (domain.User, error) {
	return s.store.ByID(ctx, id)
}

func (s *UserStore) Update(ctx context.Context, user domain.User) error {
	return s.store.Update(ctx, user)
}

var _ domain.UserRepository = (*UserStore)(nil)

type CredentialStore struct{ store *Store }

func (s *Store) Credentials() *CredentialStore { return &CredentialStore{store: s} }

func (s *Store) CreateCredential(ctx context.Context, credential domain.PasswordCredential) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.credentials[credential.UserID] = credential
	return nil
}

func (s *Store) ByUserID(ctx context.Context, userID string) (domain.PasswordCredential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.credentials[userID]
	if !ok {
		return domain.PasswordCredential{}, errors.New("credential not found")
	}
	return value, nil
}

func (s *Store) UpdateCredential(ctx context.Context, credential domain.PasswordCredential) error {
	return s.CreateCredential(ctx, credential)
}

func (s *CredentialStore) Create(ctx context.Context, credential domain.PasswordCredential) error {
	return s.store.CreateCredential(ctx, credential)
}

func (s *CredentialStore) ByUserID(ctx context.Context, userID string) (domain.PasswordCredential, error) {
	return s.store.ByUserID(ctx, userID)
}

func (s *CredentialStore) Update(ctx context.Context, credential domain.PasswordCredential) error {
	return s.store.UpdateCredential(ctx, credential)
}

var _ domain.CredentialRepository = (*CredentialStore)(nil)

type SessionStore struct{ store *Store }

func (s *Store) Sessions() *SessionStore { return &SessionStore{store: s} }

func (s *Store) CreateSession(ctx context.Context, session domain.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.TokenHash] = session
	return nil
}

func (s *Store) ByTokenHash(ctx context.Context, hash string) (domain.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.sessions[hash]
	if !ok {
		return domain.Session{}, domain.ErrSessionNotFound
	}
	return value, nil
}

func (s *Store) Rotate(ctx context.Context, oldHash string, next domain.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	old, ok := s.sessions[oldHash]
	if !ok || old.Revoked || !time.Now().Before(old.ExpiresAt) {
		return domain.ErrTokenInvalid
	}
	old.Revoked = true
	s.sessions[oldHash] = old
	s.sessions[next.TokenHash] = next
	return nil
}

func (s *Store) Revoke(ctx context.Context, hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.sessions[hash]
	if !ok {
		return domain.ErrSessionNotFound
	}
	value.Revoked = true
	s.sessions[hash] = value
	return nil
}

func (s *Store) RevokeAll(ctx context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for hash, session := range s.sessions {
		if session.UserID == userID {
			session.Revoked = true
			s.sessions[hash] = session
		}
	}
	return nil
}

func (s *SessionStore) Create(ctx context.Context, session domain.Session) error {
	return s.store.CreateSession(ctx, session)
}

func (s *SessionStore) ByTokenHash(ctx context.Context, hash string) (domain.Session, error) {
	return s.store.ByTokenHash(ctx, hash)
}

func (s *SessionStore) Rotate(ctx context.Context, oldHash string, next domain.Session) error {
	return s.store.Rotate(ctx, oldHash, next)
}

func (s *SessionStore) Revoke(ctx context.Context, hash string) error {
	return s.store.Revoke(ctx, hash)
}

func (s *SessionStore) RevokeAll(ctx context.Context, userID string) error {
	return s.store.RevokeAll(ctx, userID)
}

type VerificationTokenStore struct{ store *Store }

func (s *Store) VerificationTokens() *VerificationTokenStore {
	return &VerificationTokenStore{store: s}
}

func (s *Store) CreateVerificationToken(ctx context.Context, token domain.VerificationToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.verificationTokens[token.TokenHash] = token
	return nil
}

func (s *Store) ConsumeVerificationToken(ctx context.Context, tokenHash string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	token, ok := s.verificationTokens[tokenHash]
	if !ok || token.Used {
		return "", domain.ErrVerificationInvalid
	}
	if !time.Now().Before(token.ExpiresAt) {
		return "", domain.ErrVerificationExpired
	}
	token.Used = true
	s.verificationTokens[tokenHash] = token
	return token.UserID, nil
}

func (s *VerificationTokenStore) Create(ctx context.Context, token domain.VerificationToken) error {
	return s.store.CreateVerificationToken(ctx, token)
}

func (s *VerificationTokenStore) Consume(ctx context.Context, tokenHash string) (string, error) {
	return s.store.ConsumeVerificationToken(ctx, tokenHash)
}

var _ domain.VerificationTokenStore = (*VerificationTokenStore)(nil)

var _ domain.SessionStore = (*SessionStore)(nil)

func (s *Store) Append(ctx context.Context, event domain.OutboxEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return nil
}

var _ domain.OutboxStore = (*Store)(nil)

type Mailer struct{}

func (Mailer) SendVerification(context.Context, string, string) error  { return nil }
func (Mailer) SendPasswordReset(context.Context, string, string) error { return nil }
