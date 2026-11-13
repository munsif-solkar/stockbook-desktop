package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	defaultSupabaseURL     = "http://127.0.0.1:54321"
	defaultSupabaseAnonKey = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZS1kZW1vIiwicm9sZSI6ImFub24iLCJleHAiOjE5ODM4MTI5OTZ9.CRXP1A7WOeoJeXxjNni43kdQwgnWNReilDMblYTn_I0"
	apiTimeout             = 25 * time.Second
	maxResponseBody        = 32 << 20
	configFileName         = "config.json"
)

// backendConfig is an optional config.json in the Stockbook data directory.
// It lets a packaged desktop build point at a hosted Supabase project without
// environment variables. The anon key is public by design; never put the
// service-role key here.
type backendConfig struct {
	SupabaseURL     string `json:"supabaseUrl"`
	SupabaseAnonKey string `json:"supabaseAnonKey"`
}

func readBackendConfig(dir string) backendConfig {
	if dir == "" {
		return backendConfig{}
	}
	data, err := os.ReadFile(filepath.Join(dir, configFileName))
	if err != nil {
		return backendConfig{}
	}
	var config backendConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return backendConfig{}
	}
	return config
}

var (
	errUnauthorized      = errors.New("unauthorized")
	errEmailConfirmation = errors.New("email confirmation required")
)

// tokenState is the Supabase session kept on disk between launches.
type tokenState struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type authSession struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	User         struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
}

func (s authSession) tokens() tokenState {
	return tokenState{
		AccessToken:  s.AccessToken,
		RefreshToken: s.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(s.ExpiresIn) * time.Second),
	}
}

type apiError struct {
	Status  int
	Code    string
	Message string
}

func (e *apiError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("request failed with status %d", e.Status)
}

func parseAPIError(status int, body []byte) error {
	var payload struct {
		Message          string `json:"message"`
		Msg              string `json:"msg"`
		ErrorDescription string `json:"error_description"`
		Error            string `json:"error"`
		ErrorCode        string `json:"error_code"`
	}
	_ = json.Unmarshal(body, &payload)

	message := payload.Msg
	if message == "" {
		message = payload.Message
	}
	if message == "" {
		message = payload.ErrorDescription
	}
	if message == "" {
		message = payload.Error
	}
	code := payload.ErrorCode
	if code == "" {
		code = payload.Error
	}
	if message == "" {
		message = strings.TrimSpace(string(body))
	}
	return &apiError{Status: status, Code: code, Message: message}
}

// supabaseAPI talks to Supabase Auth and PostgREST. Business data only moves
// through the stockbook_* RPCs, so the app never trusts client-side table
// writes for stock or money.
type supabaseAPI struct {
	baseURL string
	anonKey string
	client  *http.Client

	mu           sync.Mutex
	accessToken  string
	refreshToken string
	expiresAt    time.Time
	onTokens     func(tokenState)
}

func newSupabaseAPI(config backendConfig) (*supabaseAPI, error) {
	baseURL := strings.TrimSpace(os.Getenv("STOCKBOOK_SUPABASE_URL"))
	if baseURL == "" {
		baseURL = strings.TrimSpace(config.SupabaseURL)
	}
	if baseURL == "" {
		baseURL = defaultSupabaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	anonKey := strings.TrimSpace(os.Getenv("STOCKBOOK_SUPABASE_ANON_KEY"))
	if anonKey == "" {
		anonKey = strings.TrimSpace(config.SupabaseAnonKey)
	}
	if anonKey == "" {
		anonKey = defaultSupabaseAnonKey
	}

	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("STOCKBOOK_SUPABASE_URL is not a valid http(s) URL")
	}
	if anonKey == "" {
		return nil, fmt.Errorf("STOCKBOOK_SUPABASE_ANON_KEY is empty")
	}

	return &supabaseAPI{
		baseURL: baseURL,
		anonKey: anonKey,
		client:  &http.Client{Timeout: apiTimeout},
	}, nil
}

func (s *supabaseAPI) request(ctx context.Context, method, path string, body any, token string) (int, []byte, error) {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, s.baseURL+path, reader)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("apikey", s.anonKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, data, nil
}

func (s *supabaseAPI) session() tokenState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return tokenState{
		AccessToken:  s.accessToken,
		RefreshToken: s.refreshToken,
		ExpiresAt:    s.expiresAt,
	}
}

func (s *supabaseAPI) setSession(state tokenState) {
	s.mu.Lock()
	s.accessToken = state.AccessToken
	s.refreshToken = state.RefreshToken
	s.expiresAt = state.ExpiresAt
	callback := s.onTokens
	s.mu.Unlock()

	if callback != nil {
		callback(state)
	}
}

func (s *supabaseAPI) restoreSession(state tokenState) {
	s.mu.Lock()
	s.accessToken = state.AccessToken
	s.refreshToken = state.RefreshToken
	s.expiresAt = state.ExpiresAt
	s.mu.Unlock()
}

func (s *supabaseAPI) clearSession() {
	s.setSession(tokenState{})
}

func (s *supabaseAPI) ensureToken(ctx context.Context) (string, error) {
	s.mu.Lock()
	token := s.accessToken
	expiresAt := s.expiresAt
	s.mu.Unlock()

	if token == "" {
		return "", errUnauthorized
	}
	if expiresAt.IsZero() || time.Now().Add(60*time.Second).Before(expiresAt) {
		return token, nil
	}
	return s.refreshNow(ctx)
}

func (s *supabaseAPI) refreshNow(ctx context.Context) (string, error) {
	refresh := s.session().RefreshToken
	if refresh == "" {
		s.clearSession()
		return "", errUnauthorized
	}

	status, body, err := s.request(ctx, http.MethodPost,
		"/auth/v1/token?grant_type=refresh_token",
		map[string]any{"refresh_token": refresh}, "")
	if err != nil {
		return "", err
	}
	if status < 200 || status >= 300 {
		apiErr := parseAPIError(status, body)
		if status == 400 || status == 401 {
			s.clearSession()
			return "", errUnauthorized
		}
		return "", apiErr
	}

	var session authSession
	if err := json.Unmarshal(body, &session); err != nil {
		return "", err
	}
	if session.AccessToken == "" {
		s.clearSession()
		return "", errUnauthorized
	}
	s.setSession(session.tokens())
	return session.AccessToken, nil
}

// rpc calls a PostgREST function and decodes its JSON result.
func (s *supabaseAPI) rpc(ctx context.Context, function string, args any, out any) error {
	if args == nil {
		args = map[string]any{}
	}
	token, err := s.ensureToken(ctx)
	if err != nil {
		return err
	}

	status, body, err := s.request(ctx, http.MethodPost, "/rest/v1/rpc/"+function, args, token)
	if err != nil {
		return err
	}
	if status == 401 {
		token, err = s.refreshNow(ctx)
		if err != nil {
			return err
		}
		status, body, err = s.request(ctx, http.MethodPost, "/rest/v1/rpc/"+function, args, token)
		if err != nil {
			return err
		}
	}
	if status == 401 || status == 403 {
		s.clearSession()
		return errUnauthorized
	}
	if status < 200 || status >= 300 {
		return parseAPIError(status, body)
	}

	if out != nil && len(bytes.TrimSpace(body)) > 0 {
		if err := json.Unmarshal(body, out); err != nil {
			return fmt.Errorf("decode %s response: %w", function, err)
		}
	}
	return nil
}

func (s *supabaseAPI) authSignUp(ctx context.Context, email, password, companyName string) (tokenState, error) {
	payload := map[string]any{
		"email":    email,
		"password": password,
		"data":     map[string]any{"company_name": companyName},
	}
	status, body, err := s.request(ctx, http.MethodPost, "/auth/v1/signup", payload, "")
	if err != nil {
		return tokenState{}, err
	}
	if status < 200 || status >= 300 {
		return tokenState{}, parseAPIError(status, body)
	}

	var session authSession
	if err := json.Unmarshal(body, &session); err != nil {
		return tokenState{}, err
	}
	if session.AccessToken == "" || session.RefreshToken == "" {
		return tokenState{}, errEmailConfirmation
	}
	return session.tokens(), nil
}

func (s *supabaseAPI) authSignIn(ctx context.Context, email, password string) (tokenState, error) {
	status, body, err := s.request(ctx, http.MethodPost,
		"/auth/v1/token?grant_type=password",
		map[string]any{"email": email, "password": password}, "")
	if err != nil {
		return tokenState{}, err
	}
	if status < 200 || status >= 300 {
		return tokenState{}, parseAPIError(status, body)
	}

	var session authSession
	if err := json.Unmarshal(body, &session); err != nil {
		return tokenState{}, err
	}
	if session.AccessToken == "" || session.RefreshToken == "" {
		return tokenState{}, errUnauthorized
	}
	return session.tokens(), nil
}

func (s *supabaseAPI) authLogout(ctx context.Context, token string) {
	if token == "" {
		return
	}
	_, _, _ = s.request(ctx, http.MethodPost, "/auth/v1/logout", nil, token)
}

func (s *supabaseAPI) bootstrapCompany(ctx context.Context, name string) (*Company, error) {
	args := map[string]any{"p_name": nil}
	if trimmed := strings.TrimSpace(name); trimmed != "" {
		args["p_name"] = trimmed
	}

	var company Company
	if err := s.rpc(ctx, "stockbook_bootstrap_company", args, &company); err != nil {
		return nil, err
	}
	if company.ID == 0 {
		return nil, errors.New("could not load the workspace")
	}
	return &company, nil
}

// dataDirectory is where the app keeps its session file. Business data now
// lives in Supabase; this folder no longer holds a database.
func dataDirectory() (string, error) {
	if configured := os.Getenv("STOCKBOOK_DATA_DIR"); configured != "" {
		return filepath.Abs(configured)
	}
	return defaultDataDirectory()
}
