package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

const sessionFileName = "session.json"

func (a *App) context() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}

func (a *App) sessionFilePath() string {
	if a.dataDir == "" {
		return ""
	}
	return filepath.Join(a.dataDir, sessionFileName)
}

// writeSessionFile stores the Supabase tokens with owner-only permissions.
// It is called whenever the client rotates its tokens.
func (a *App) writeSessionFile(state tokenState) {
	path := a.sessionFilePath()
	if path == "" {
		return
	}
	if state.AccessToken == "" {
		_ = os.Remove(path)
		return
	}

	data, err := json.Marshal(state)
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return
	}
	temp := path + ".tmp"
	if err := os.WriteFile(temp, data, 0o600); err != nil {
		return
	}
	if err := os.Rename(temp, path); err != nil {
		_ = os.Remove(temp)
		return
	}
	_ = os.Chmod(path, 0o600)
}

func (a *App) loadSessionFile() (tokenState, bool) {
	path := a.sessionFilePath()
	if path == "" {
		return tokenState{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return tokenState{}, false
	}
	var state tokenState
	if err := json.Unmarshal(data, &state); err != nil || state.RefreshToken == "" {
		_ = os.Remove(path)
		return tokenState{}, false
	}
	return state, true
}

func (a *App) removeSessionFile() {
	if path := a.sessionFilePath(); path != "" {
		_ = os.Remove(path)
	}
}

// restoreSession revalidates the saved Supabase session at startup. A stale or
// revoked session simply lands on the sign-in screen; a backend that is down
// does not lock the app behind a fatal error.
func (a *App) restoreSession() {
	if a.currentMode() == modeLocal {
		a.restoreLocalSession()
		return
	}

	state, ok := a.loadSessionFile()
	if !ok {
		return
	}

	a.api.restoreSession(state)
	if time.Now().Add(60 * time.Second).After(state.ExpiresAt) {
		if _, err := a.api.refreshNow(a.context()); err != nil {
			if errors.Is(err, errUnauthorized) {
				a.removeSessionFile()
			}
			return
		}
	}

	company, err := a.api.bootstrapCompany(a.context(), "")
	if err != nil {
		if errors.Is(err, errUnauthorized) {
			a.api.clearSession()
			a.removeSessionFile()
		}
		return
	}
	a.mu.Lock()
	a.company = company
	a.mu.Unlock()
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

func (a *App) setCompany(company *Company) {
	a.mu.Lock()
	a.company = company
	a.mu.Unlock()
}

// destroySession revokes the Supabase token when possible and always clears
// the local session.
func (a *App) destroySession() {
	state := a.api.session()
	a.api.clearSession()
	a.setCompany(nil)

	if state.AccessToken != "" {
		ctx, cancel := context.WithTimeout(a.context(), 5*time.Second)
		defer cancel()
		a.api.authLogout(ctx, state.AccessToken)
	}
	a.removeSessionFile()
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
