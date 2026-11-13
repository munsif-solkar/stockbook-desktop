package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadBackendConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, configFileName)
	content := `{"supabaseUrl":"https://example.supabase.co","supabaseAnonKey":"file-key"}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	config := readBackendConfig(dir)
	if config.SupabaseURL != "https://example.supabase.co" {
		t.Errorf("url = %q", config.SupabaseURL)
	}
	if config.SupabaseAnonKey != "file-key" {
		t.Errorf("anon key = %q", config.SupabaseAnonKey)
	}
}

func TestReadBackendConfigIgnoresMissingOrInvalidFile(t *testing.T) {
	if config := readBackendConfig(t.TempDir()); config != (backendConfig{}) {
		t.Errorf("missing file returned %+v", config)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, configFileName), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if config := readBackendConfig(dir); config != (backendConfig{}) {
		t.Errorf("invalid file returned %+v", config)
	}
}

func TestBackendConfigPrecedence(t *testing.T) {
	dir := t.TempDir()
	content := `{"supabaseUrl":"https://file.supabase.co","supabaseAnonKey":"file-key"}`
	if err := os.WriteFile(filepath.Join(dir, configFileName), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("STOCKBOOK_SUPABASE_URL", "")
	t.Setenv("STOCKBOOK_SUPABASE_ANON_KEY", "")

	api, err := newSupabaseAPI(readBackendConfig(dir))
	if err != nil {
		t.Fatal(err)
	}
	if api.baseURL != "https://file.supabase.co" || api.anonKey != "file-key" {
		t.Errorf("config file not used: %q %q", api.baseURL, api.anonKey)
	}

	t.Setenv("STOCKBOOK_SUPABASE_URL", "https://env.supabase.co")
	t.Setenv("STOCKBOOK_SUPABASE_ANON_KEY", "env-key")

	api, err = newSupabaseAPI(readBackendConfig(dir))
	if err != nil {
		t.Fatal(err)
	}
	if api.baseURL != "https://env.supabase.co" || api.anonKey != "env-key" {
		t.Errorf("environment did not override config: %q %q", api.baseURL, api.anonKey)
	}
}

func TestBackendConfigRejectsInvalidURL(t *testing.T) {
	t.Setenv("STOCKBOOK_SUPABASE_URL", "ftp://example.com")
	t.Setenv("STOCKBOOK_SUPABASE_ANON_KEY", "key")
	if _, err := newSupabaseAPI(backendConfig{}); err == nil {
		t.Fatal("expected invalid URL to be rejected")
	}
}
