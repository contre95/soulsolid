package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_EnvVarAndEnvFileTags(t *testing.T) {
	t.Setenv("SOULSOLID_TEST_TOKEN", "token-from-env")

	secretFile := filepath.Join(t.TempDir(), "acoustid_secret")
	if err := os.WriteFile(secretFile, []byte("secret-from-file\n"), 0o600); err != nil {
		t.Fatalf("failed to write secret file: %v", err)
	}
	t.Setenv("SOULSOLID_TEST_SECRET_FILE", secretFile)

	configYAML := `
libraryPath: ./music
downloadPath: ./downloads
database:
  path: ./library.db
telegram:
  token: !env_var SOULSOLID_TEST_TOKEN
metadata:
  providers:
    acoustid:
      enabled: true
      secret: !env_file SOULSOLID_TEST_SECRET_FILE
`
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte(configYAML), 0o600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	m := &Manager{}
	cfg, err := m.loadConfig(configPath)
	if err != nil {
		t.Fatalf("loadConfig returned error: %v", err)
	}

	if cfg.Telegram.Token != "token-from-env" {
		t.Errorf("expected telegram token %q, got %q", "token-from-env", cfg.Telegram.Token)
	}

	acoustid, ok := cfg.Metadata.Providers["acoustid"]
	if !ok {
		t.Fatalf("expected acoustid provider to be present")
	}
	if acoustid.Secret == nil || *acoustid.Secret != "secret-from-file" {
		t.Errorf("expected acoustid secret %q, got %v", "secret-from-file", acoustid.Secret)
	}
}

func TestLoadConfig_EnvFileMissingFile(t *testing.T) {
	t.Setenv("SOULSOLID_TEST_MISSING_FILE", filepath.Join(t.TempDir(), "does-not-exist"))

	configYAML := `
libraryPath: ./music
downloadPath: ./downloads
database:
  path: ./library.db
telegram:
  token: !env_file SOULSOLID_TEST_MISSING_FILE
`
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte(configYAML), 0o600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	m := &Manager{}
	if _, err := m.loadConfig(configPath); err == nil {
		t.Fatal("expected error when the file referenced by !env_file does not exist")
	}
}
