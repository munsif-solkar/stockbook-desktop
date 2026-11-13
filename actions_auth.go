package main

import (
	"errors"
	"strings"
)

func (a *App) Session() SessionResult {
	if a.initErr != nil {
		return SessionResult{Error: a.initErr.Error()}
	}
	company := a.currentCompany()
	return SessionResult{Authenticated: company != nil, Company: company}
}

func isAlreadyRegistered(err error) bool {
	var apiErr *apiError
	if !errors.As(err, &apiErr) {
		return false
	}
	text := strings.ToLower(apiErr.Code + " " + apiErr.Message)
	return strings.Contains(text, "already registered") ||
		strings.Contains(text, "already exists") ||
		strings.Contains(text, "user_already_exists")
}

func isInvalidCredentials(err error) bool {
	var apiErr *apiError
	if !errors.As(err, &apiErr) {
		return false
	}
	text := strings.ToLower(apiErr.Code + " " + apiErr.Message)
	return strings.Contains(text, "invalid login credentials") ||
		strings.Contains(text, "invalid_credentials") ||
		strings.Contains(text, "invalid grant")
}

func isEmailNotConfirmed(err error) bool {
	var apiErr *apiError
	if !errors.As(err, &apiErr) {
		return false
	}
	text := strings.ToLower(apiErr.Code + " " + apiErr.Message)
	return strings.Contains(text, "email not confirmed") ||
		strings.Contains(text, "email_not_confirmed")
}

func friendlyAuthError(err error) string {
	var apiErr *apiError
	if errors.As(err, &apiErr) {
		switch {
		case apiErr.Status == 429:
			return "Too many attempts. Wait a few minutes and try again."
		case apiErr.Status >= 500:
			return "The Stockbook backend is unavailable. Please try again in a moment."
		case apiErr.Message != "":
			return apiErr.Message
		}
	}
	return "Could not reach the Stockbook backend. Check your connection and that Supabase is running."
}

func (a *App) Register(in RegisterInput) AuthResult {
	if a.initErr != nil {
		return AuthResult{Error: a.initErr.Error()}
	}

	valid, fieldErrors := validateRegister(in)
	if fieldErrors != nil {
		return AuthResult{FieldErrors: fieldErrors, Company: nil}
	}

	state, err := a.api.authSignUp(a.context(), valid.Email, valid.Password, valid.CompanyName)
	if err != nil {
		if isAlreadyRegistered(err) {
			return AuthResult{FieldErrors: map[string][]string{
				"email": {"An account with this email already exists."},
			}}
		}
		if errors.Is(err, errEmailConfirmation) {
			return AuthResult{Error: "Check your email to confirm the account, then sign in."}
		}
		return AuthResult{Error: friendlyAuthError(err)}
	}

	a.api.setSession(state)

	company, err := a.api.bootstrapCompany(a.context(), valid.CompanyName)
	if err != nil {
		return AuthResult{Error: "Your account was created, but the workspace could not be loaded. Please sign in."}
	}

	a.setCompany(company)
	return AuthResult{Company: company}
}

func (a *App) Login(in LoginInput) AuthResult {
	if a.initErr != nil {
		return AuthResult{Error: a.initErr.Error()}
	}

	valid, fieldErrors := validateLogin(in)
	if fieldErrors != nil {
		return AuthResult{FieldErrors: fieldErrors}
	}

	if a.loginBlocked(valid.Email) {
		return AuthResult{Error: "Too many sign-in attempts. Try again in a few minutes."}
	}

	state, err := a.api.authSignIn(a.context(), valid.Email, valid.Password)
	if err != nil {
		a.recordLoginFailure(valid.Email)
		if isEmailNotConfirmed(err) {
			return AuthResult{Error: "Confirm your email address before signing in. Check your inbox for the confirmation link."}
		}
		if isInvalidCredentials(err) {
			return AuthResult{Error: "Incorrect email or password."}
		}
		return AuthResult{Error: friendlyAuthError(err)}
	}

	a.api.setSession(state)

	company, err := a.api.bootstrapCompany(a.context(), "")
	if err != nil {
		a.api.clearSession()
		return AuthResult{Error: friendlyAuthError(err)}
	}

	a.clearLoginFailures(valid.Email)
	a.setCompany(company)
	return AuthResult{Company: company}
}

func (a *App) Logout() error {
	a.destroySession()
	return nil
}
