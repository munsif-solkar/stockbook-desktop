package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/scrypt"
)

const (
	sessionDays = 7
	scryptN     = 16384
	scryptR     = 8
	scryptP     = 1
	scryptKey   = 64
)

var (
	errUnauthorized = errors.New("unauthorized")
	dummyHashOnce   sync.Once
	dummyHashValue  string
)

func randomToken(bytes int) (string, error) {
	buffer := make([]byte, bytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func hashPassword(password string) (string, error) {
	salt, err := randomToken(16)
	if err != nil {
		return "", err
	}
	derived, err := scrypt.Key([]byte(password), []byte(salt), scryptN, scryptR, scryptP, scryptKey)
	if err != nil {
		return "", err
	}
	return "scrypt$" + strconv.Itoa(scryptN) + "$" + strconv.Itoa(scryptR) + "$" + strconv.Itoa(scryptP) + "$" + salt + "$" + base64.RawURLEncoding.EncodeToString(derived), nil
}

func verifyPassword(password, stored string) bool {
	parts := strings.Split(stored, "$")
	if len(parts) != 6 || parts[0] != "scrypt" {
		return false
	}
	n, errN := strconv.Atoi(parts[1])
	r, errR := strconv.Atoi(parts[2])
	p, errP := strconv.Atoi(parts[3])
	if errN != nil || errR != nil || errP != nil || n <= 0 || r <= 0 || p <= 0 {
		return false
	}
	expected, err := base64.RawURLEncoding.DecodeString(parts[5])
	if err != nil || len(expected) == 0 {
		return false
	}
	derived, err := scrypt.Key([]byte(password), []byte(parts[4]), n, r, p, len(expected))
	if err != nil {
		return false
	}
	return len(derived) == len(expected) && subtle.ConstantTimeCompare(derived, expected) == 1
}

func dummyHash() string {
	dummyHashOnce.Do(func() {
		dummyHashValue, _ = hashPassword("not-a-real-password-1")
	})
	return dummyHashValue
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func (a *App) sessionFilePath() string {
	return filepath.Join(filepath.Dir(a.dbPath), "session")
}

func (a *App) createSession(companyID int64) error {
	token, err := randomToken(32)
	if err != nil {
		return err
	}
	now := time.Now()
	expiresAt := now.Add(sessionDays * 24 * time.Hour)
	if _, err := a.db.Exec(
		"INSERT INTO sessions (company_id, token_hash, expires_at, created_at) VALUES (?, ?, ?, ?)",
		companyID, hashToken(token), jsISO(expiresAt), jsISO(now),
	); err != nil {
		return err
	}
	path := a.sessionFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	// Write through a temporary file so a crash can never leave a truncated
	// session behind.
	temp := path + ".tmp"
	if err := os.WriteFile(temp, []byte(token), 0o600); err != nil {
		return err
	}
	if err := os.Rename(temp, path); err != nil {
		_ = os.Remove(temp)
		return err
	}
	_ = os.Chmod(path, 0o600)

	a.mu.Lock()
	a.token = token
	a.mu.Unlock()
	return nil
}

func (a *App) restoreSession() {
	data, err := os.ReadFile(a.sessionFilePath())
	if err != nil {
		return
	}
	token := strings.TrimSpace(string(data))
	if token == "" {
		return
	}
	company, err := a.companyForToken(token)
	if err != nil || company == nil {
		_ = os.Remove(a.sessionFilePath())
		return
	}
	a.mu.Lock()
	a.token = token
	a.company = company
	a.mu.Unlock()
}

func (a *App) destroySession() {
	a.mu.Lock()
	token := a.token
	a.token = ""
	a.company = nil
	a.mu.Unlock()

	if token != "" && a.db != nil {
		if _, err := a.db.Exec("DELETE FROM sessions WHERE token_hash = ?", hashToken(token)); err != nil {
			_ = err
		}
	}
	if err := os.Remove(a.sessionFilePath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		_ = err
	}
}

func (a *App) companyForToken(token string) (*Company, error) {
	var company Company
	err := a.db.QueryRow(
		`SELECT c.id, c.name, c.email
		 FROM sessions s
		 JOIN companies c ON c.id = s.company_id
		 WHERE s.token_hash = ? AND s.expires_at > ?
		 LIMIT 1`,
		hashToken(token), nowISO(),
	).Scan(&company.ID, &company.Name, &company.Email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &company, nil
}

func (a *App) currentCompany() *Company {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.company == nil {
		return nil
	}
	copy := *a.company
	return &copy
}

func (a *App) requireCompany() (*Company, error) {
	company := a.currentCompany()
	if company == nil {
		return nil, errUnauthorized
	}
	return company, nil
}

type attemptEntry struct {
	count   int
	resetAt time.Time
}

func (a *App) loginBlocked(email string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	entry, ok := a.attempts[email]
	if !ok {
		return false
	}
	if !entry.resetAt.After(time.Now()) {
		delete(a.attempts, email)
		return false
	}
	return entry.count >= 8
}

func (a *App) recordLoginFailure(email string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now()
	entry, ok := a.attempts[email]
	if !ok || !entry.resetAt.After(now) {
		a.attempts[email] = &attemptEntry{count: 1, resetAt: now.Add(15 * time.Minute)}
		return
	}
	entry.count++
}

func (a *App) clearLoginFailures(email string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.attempts, email)
}
