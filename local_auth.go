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
	dummyHashOnce  sync.Once
	dummyHashValue string
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

func (a *App) localSessionFilePath() string {
	if a.local == nil || a.local.path == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(a.local.path), "session")
}

func (a *App) localCreateSession(companyID int64) error {
	token, err := randomToken(32)
	if err != nil {
		return err
	}
	now := time.Now()
	expiresAt := now.Add(sessionDays * 24 * time.Hour)
	if _, err := a.local.db.Exec(
		"INSERT INTO sessions (company_id, token_hash, expires_at, created_at) VALUES (?, ?, ?, ?)",
		companyID, hashToken(token), jsISO(expiresAt), jsISO(now),
	); err != nil {
		return err
	}
	path := a.localSessionFilePath()
	if path == "" {
		return errors.New("local session path unavailable")
	}
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
	a.localToken = token
	a.mu.Unlock()
	return nil
}

func (a *App) restoreLocalSession() {
	if a.local == nil {
		return
	}
	path := a.localSessionFilePath()
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	token := strings.TrimSpace(string(data))
	if token == "" {
		return
	}
	company, err := a.localCompanyForToken(token)
	if err != nil || company == nil {
		_ = os.Remove(path)
		return
	}
	a.mu.Lock()
	a.localToken = token
	a.company = company
	a.mu.Unlock()
}

func (a *App) destroyLocalSession() {
	a.mu.Lock()
	token := a.localToken
	a.localToken = ""
	a.company = nil
	a.mu.Unlock()

	if token != "" && a.local != nil && a.local.db != nil {
		if _, err := a.local.db.Exec("DELETE FROM sessions WHERE token_hash = ?", hashToken(token)); err != nil {
			_ = err
		}
	}
	if path := a.localSessionFilePath(); path != "" {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			_ = err
		}
	}
}

func (a *App) localCompanyForToken(token string) (*Company, error) {
	var company Company
	err := a.local.db.QueryRow(
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

func (a *App) localRegister(in RegisterInput) AuthResult {
	if a.local == nil {
		return AuthResult{Error: "Could not open the local database."}
	}

	valid, fieldErrors := validateRegister(in)
	if fieldErrors != nil {
		return AuthResult{FieldErrors: fieldErrors, Company: nil}
	}

	passwordHash, err := hashPassword(valid.Password)
	if err != nil {
		return AuthResult{Error: "We could not create your account. Please try again."}
	}

	now := nowISO()
	result, err := a.local.db.Exec(
		"INSERT INTO companies (name, email, password_hash, created_at) VALUES (?, ?, ?, ?)",
		valid.CompanyName, valid.Email, passwordHash, now,
	)
	if err != nil {
		if isUniqueConstraint(err) {
			return AuthResult{FieldErrors: map[string][]string{
				"email": {"An account with this email already exists."},
			}}
		}
		return AuthResult{Error: "We could not create your account. Please try again."}
	}

	companyID, err := result.LastInsertId()
	if err != nil {
		return AuthResult{Error: "We could not create your account. Please try again."}
	}

	if err := a.localCreateSession(companyID); err != nil {
		return AuthResult{Error: "We could not create your account. Please try again."}
	}

	company := &Company{ID: companyID, Name: valid.CompanyName, Email: valid.Email}
	a.mu.Lock()
	a.company = company
	a.mu.Unlock()

	return AuthResult{Company: company}
}

func (a *App) localLogin(in LoginInput) AuthResult {
	if a.local == nil {
		return AuthResult{Error: "Could not open the local database."}
	}

	valid, fieldErrors := validateLogin(in)
	if fieldErrors != nil {
		return AuthResult{FieldErrors: fieldErrors}
	}

	if a.loginBlocked(valid.Email) {
		return AuthResult{Error: "Too many sign-in attempts. Try again in a few minutes."}
	}

	var companyID int64
	var storedHash string
	err := a.local.db.QueryRow(
		"SELECT id, password_hash FROM companies WHERE email = ?", valid.Email,
	).Scan(&companyID, &storedHash)
	if errors.Is(err, sql.ErrNoRows) {
		verifyPassword(valid.Password, dummyHash())
		a.recordLoginFailure(valid.Email)
		return AuthResult{Error: "Incorrect email or password."}
	}
	if err != nil {
		return AuthResult{Error: "Incorrect email or password."}
	}

	if !verifyPassword(valid.Password, storedHash) {
		a.recordLoginFailure(valid.Email)
		return AuthResult{Error: "Incorrect email or password."}
	}

	var company Company
	if err := a.local.db.QueryRow(
		"SELECT id, name, email FROM companies WHERE id = ?", companyID,
	).Scan(&company.ID, &company.Name, &company.Email); err != nil {
		return AuthResult{Error: "Incorrect email or password."}
	}

	if err := a.localCreateSession(company.ID); err != nil {
		return AuthResult{Error: "We could not sign you in. Please try again."}
	}

	a.clearLoginFailures(valid.Email)
	a.mu.Lock()
	a.company = &company
	a.mu.Unlock()

	return AuthResult{Company: &company}
}

func isUniqueConstraint(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique constraint failed") ||
		strings.Contains(message, "constraint failed: unique")
}
