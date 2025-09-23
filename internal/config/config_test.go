package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatalf("write config %s: %v", name, err)
	}
}

func TestLoadConfigWithDefaults(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "default.yaml", `
app:
  name: test-app
server:
  host: 127.0.0.1
  port: 9090
database:
  driver: sqlite
  dsn: file:./test.db
redis:
  addr: 127.0.0.1:6379
auth:
  accessTokenSecret: "abcdefghijklmnopqrstuvwxyz123456"
  refreshTokenSecret: "abcdefghijklmnopqrstuvwxyz1234567890"
  accessTokenTTL: 15m
  refreshTokenTTL: 720h
  apiKeyHashSecret: "abcdefghijklmnopqrstuvwxyz098765"
logging:
  level: debug
`)

	cfg, err := Load(dir, "")
	if err != nil {
		t.Fatalf("load config failed: %v", err)
	}

	if cfg.Server.MaxRequestBody != 3*1024*1024 {
		t.Fatalf("expected default max request body 3MB got %d", cfg.Server.MaxRequestBody)
	}
	if cfg.Logging.Level != "debug" {
		t.Fatalf("expected logging level debug got %s", cfg.Logging.Level)
	}
	if got := cfg.Server.CORS.AllowOrigins; len(got) != 1 || got[0] != "*" {
		t.Fatalf("expected default CORS allow origins to be ['*'] got %#v", got)
	}
	if !cfg.Server.SecurityHeaders.ContentTypeNosniff {
		t.Fatalf("expected default content type nosniff to be true")
	}
}

func TestLoadConfigInvalidSecrets(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "default.yaml", `
app:
  name: test-app
database:
  driver: sqlite
redis:
  addr: 127.0.0.1:6379
auth:
  accessTokenSecret: short
  refreshTokenSecret: short
  accessTokenTTL: 15m
  refreshTokenTTL: 720h
  apiKeyHashSecret: short
`)

	if _, err := Load(dir, ""); err == nil {
		t.Fatalf("expected error for weak secrets")
	}
}

func TestLoadConfigCustomRequestBody(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "default.yaml", `
app:
  name: test-app
server:
  maxRequestBody: 5242880
database:
  driver: sqlite
redis:
  addr: 127.0.0.1:6379
auth:
  accessTokenSecret: "abcdefghijklmnopqrstuvwxyz123456"
  refreshTokenSecret: "abcdefghijklmnopqrstuvwxyz1234567890"
  accessTokenTTL: 15m
  refreshTokenTTL: 720h
  apiKeyHashSecret: "abcdefghijklmnopqrstuvwxyz098765"
`)

	cfg, err := Load(dir, "")
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if cfg.Server.MaxRequestBody != 5*1024*1024 {
		t.Fatalf("expected 5MB got %d", cfg.Server.MaxRequestBody)
	}
}

func TestLoadConfigRejectsWildcardOriginInProd(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "default.yaml", `
app:
  name: test-app
  env: production
server:
  cors:
    allowOrigins:
      - "*"
database:
  driver: sqlite
redis:
  addr: 127.0.0.1:6379
auth:
  accessTokenSecret: "abcdefghijklmnopqrstuvwxyz123456"
  refreshTokenSecret: "abcdefghijklmnopqrstuvwxyz1234567890"
  accessTokenTTL: 15m
  refreshTokenTTL: 720h
  apiKeyHashSecret: "abcdefghijklmnopqrstuvwxyz098765"
`)

	if _, err := Load(dir, ""); err == nil {
		t.Fatalf("expected wildcard origins to be rejected in production")
	}
}

func TestLoadConfigSeedAdminFromConfig(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "default.yaml", `
app:
  name: test-app
server:
  host: 0.0.0.0
database:
  driver: sqlite
redis:
  addr: 127.0.0.1:6379
auth:
  accessTokenSecret: "abcdefghijklmnopqrstuvwxyz123456"
  refreshTokenSecret: "abcdefghijklmnopqrstuvwxyz1234567890"
  accessTokenTTL: 15m
  refreshTokenTTL: 720h
  apiKeyHashSecret: "abcdefghijklmnopqrstuvwxyz098765"
seed:
  admin:
    email: admin@example.com
    password: super-secret-password
    role: editor
`)

	cfg, err := Load(dir, "")
	if err != nil {
		t.Fatalf("load config failed: %v", err)
	}

	if cfg.Seed.Admin.Email != "admin@example.com" {
		t.Fatalf("expected seed admin email from config got %s", cfg.Seed.Admin.Email)
	}
	if cfg.Seed.Admin.Password != "super-secret-password" {
		t.Fatalf("expected seed admin password from config")
	}
	if cfg.Seed.Admin.Role != "editor" {
		t.Fatalf("expected seed admin role editor got %s", cfg.Seed.Admin.Role)
	}
}
