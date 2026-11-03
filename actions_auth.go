package main

import (
	"database/sql"
	"errors"
	"strings"
)

func (a *App) databaseError() error {
	if a.db == nil {
		if a.dbErr != nil {
			return a.dbErr
		}
		return errors.New("database unavailable")
	}
	return nil
}

func (a *App) Session() SessionResult {
	if err := a.databaseError(); err != nil {
		return SessionResult{Error: "Could not open the local database."}
	}
	company := a.currentCompany()
	return SessionResult{Authenticated: company != nil, Company: company}
}

func (a *App) Register(in RegisterInput) AuthResult {
	if err := a.databaseError(); err != nil {
		return AuthResult{Error: "Could not open the local database."}
	}

	valid, fieldErrors := validateRegister(in)
	if fieldErrors != nil {
		return AuthResult{
			FieldErrors: fieldErrors,
			Company:     nil,
		}
	}

	passwordHash, err := hashPassword(valid.Password)
	if err != nil {
		return AuthResult{Error: "We could not create your account. Please try again."}
	}

	now := nowISO()
	result, err := a.db.Exec(
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

	if err := a.createSession(companyID); err != nil {
		return AuthResult{Error: "We could not create your account. Please try again."}
	}

	company := &Company{ID: companyID, Name: valid.CompanyName, Email: valid.Email}
	a.mu.Lock()
	a.company = company
	a.mu.Unlock()

	return AuthResult{Company: company}
}

func (a *App) Login(in LoginInput) AuthResult {
	if err := a.databaseError(); err != nil {
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
	err := a.db.QueryRow(
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
	if err := a.db.QueryRow(
		"SELECT id, name, email FROM companies WHERE id = ?", companyID,
	).Scan(&company.ID, &company.Name, &company.Email); err != nil {
		return AuthResult{Error: "Incorrect email or password."}
	}

	if err := a.createSession(company.ID); err != nil {
		return AuthResult{Error: "We could not sign you in. Please try again."}
	}

	a.clearLoginFailures(valid.Email)
	a.mu.Lock()
	a.company = &company
	a.mu.Unlock()

	return AuthResult{Company: &company}
}

func (a *App) Logout() error {
	a.destroySession()
	return nil
}

func isUniqueConstraint(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique constraint failed") ||
		strings.Contains(message, "constraint failed: unique")
}
