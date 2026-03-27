package setup

import (
	"os"
	"strings"
	"testing"
)

func TestDecideAdminBootstrap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		totalUsers int64
		adminUsers int64
		should     bool
		reason     string
	}{
		{
			name:       "empty database should create admin",
			totalUsers: 0,
			adminUsers: 0,
			should:     true,
			reason:     adminBootstrapReasonEmptyDatabase,
		},
		{
			name:       "admin exists should skip",
			totalUsers: 10,
			adminUsers: 1,
			should:     false,
			reason:     adminBootstrapReasonAdminExists,
		},
		{
			name:       "users exist without admin should skip",
			totalUsers: 5,
			adminUsers: 0,
			should:     false,
			reason:     adminBootstrapReasonUsersExistWithoutAdmin,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := decideAdminBootstrap(tc.totalUsers, tc.adminUsers)
			if got.shouldCreate != tc.should {
				t.Fatalf("shouldCreate=%v, want %v", got.shouldCreate, tc.should)
			}
			if got.reason != tc.reason {
				t.Fatalf("reason=%q, want %q", got.reason, tc.reason)
			}
		})
	}
}

func TestSetupDefaultAdminConcurrency(t *testing.T) {
	t.Run("simple mode admin uses higher concurrency", func(t *testing.T) {
		t.Setenv("RUN_MODE", "simple")
		if got := setupDefaultAdminConcurrency(); got != simpleModeAdminConcurrency {
			t.Fatalf("setupDefaultAdminConcurrency()=%d, want %d", got, simpleModeAdminConcurrency)
		}
	})

	t.Run("standard mode keeps existing default", func(t *testing.T) {
		t.Setenv("RUN_MODE", "standard")
		if got := setupDefaultAdminConcurrency(); got != defaultUserConcurrency {
			t.Fatalf("setupDefaultAdminConcurrency()=%d, want %d", got, defaultUserConcurrency)
		}
	})
}

func TestWriteConfigFileKeepsDefaultUserConcurrency(t *testing.T) {
	t.Setenv("RUN_MODE", "simple")
	t.Setenv("DATA_DIR", t.TempDir())

	if err := writeConfigFile(&SetupConfig{}); err != nil {
		t.Fatalf("writeConfigFile() error = %v", err)
	}

	data, err := os.ReadFile(GetConfigFilePath())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !strings.Contains(string(data), "user_concurrency: 5") {
		t.Fatalf("config missing default user concurrency, got:\n%s", string(data))
	}
}

func TestAutoSetupEnabledFromURLPair(t *testing.T) {
	t.Setenv("AUTO_SETUP", "")
	t.Setenv("DATABASE_URL", "postgres://user:pass@db.example.com:5432/sub2api")
	t.Setenv("REDIS_URL", "redis://:pass@redis.example.com:6379/0")

	if !AutoSetupEnabled() {
		t.Fatalf("AutoSetupEnabled() = false, want true when DATABASE_URL and REDIS_URL are set")
	}
}

func TestAutoSetupEnabledExplicitFalseWins(t *testing.T) {
	t.Setenv("AUTO_SETUP", "false")
	t.Setenv("DATABASE_URL", "postgres://user:pass@db.example.com:5432/sub2api")
	t.Setenv("REDIS_URL", "redis://:pass@redis.example.com:6379/0")

	if AutoSetupEnabled() {
		t.Fatalf("AutoSetupEnabled() = true, want false when AUTO_SETUP=false")
	}
}

func TestAutoSetupEnabledInvalidURLPairFallsBackToFalse(t *testing.T) {
	t.Setenv("AUTO_SETUP", "")
	t.Setenv("DATABASE_URL", "postgres://")
	t.Setenv("REDIS_URL", "redis://:pass@redis.example.com:6379/0")

	if AutoSetupEnabled() {
		t.Fatalf("AutoSetupEnabled() = true, want false for malformed DATABASE_URL")
	}
}

func TestAutoSetupEnabledExplicitTrueWinsEvenWithInvalidURLs(t *testing.T) {
	t.Setenv("AUTO_SETUP", "true")
	t.Setenv("DATABASE_URL", "postgres://")
	t.Setenv("REDIS_URL", "redis://")

	if !AutoSetupEnabled() {
		t.Fatalf("AutoSetupEnabled() = false, want true when AUTO_SETUP=true")
	}
}

func TestParseDatabaseURL(t *testing.T) {
	cfg, err := parseDatabaseURL("postgres://dbuser:dbpass@db.example.com:5433/sub2api?sslmode=require")
	if err != nil {
		t.Fatalf("parseDatabaseURL() error = %v", err)
	}

	if cfg.Host != "db.example.com" {
		t.Fatalf("Host = %q, want %q", cfg.Host, "db.example.com")
	}
	if cfg.Port != 5433 {
		t.Fatalf("Port = %d, want %d", cfg.Port, 5433)
	}
	if cfg.User != "dbuser" {
		t.Fatalf("User = %q, want %q", cfg.User, "dbuser")
	}
	if cfg.Password != "dbpass" {
		t.Fatalf("Password = %q, want %q", cfg.Password, "dbpass")
	}
	if cfg.DBName != "sub2api" {
		t.Fatalf("DBName = %q, want %q", cfg.DBName, "sub2api")
	}
	if cfg.SSLMode != "require" {
		t.Fatalf("SSLMode = %q, want %q", cfg.SSLMode, "require")
	}
}

func TestParseRedisURL(t *testing.T) {
	cfg, err := parseRedisURL("rediss://:redispass@redis.example.com:6380/2")
	if err != nil {
		t.Fatalf("parseRedisURL() error = %v", err)
	}

	if cfg.Host != "redis.example.com" {
		t.Fatalf("Host = %q, want %q", cfg.Host, "redis.example.com")
	}
	if cfg.Port != 6380 {
		t.Fatalf("Port = %d, want %d", cfg.Port, 6380)
	}
	if cfg.Password != "redispass" {
		t.Fatalf("Password = %q, want %q", cfg.Password, "redispass")
	}
	if cfg.DB != 2 {
		t.Fatalf("DB = %d, want %d", cfg.DB, 2)
	}
	if !cfg.EnableTLS {
		t.Fatalf("EnableTLS = false, want true")
	}
}

func TestParseRedisURLTLSFromQuery(t *testing.T) {
	cfg, err := parseRedisURL("redis://:redispass@redis.example.com:6379/0?ssl=true")
	if err != nil {
		t.Fatalf("parseRedisURL() error = %v", err)
	}
	if !cfg.EnableTLS {
		t.Fatalf("EnableTLS = false, want true when ssl=true")
	}
}

func TestGetRedisURLFromEnvPrefersTLSURL(t *testing.T) {
	t.Setenv("REDIS_URL", "redis://:p@redis.example.com:6379/0")
	t.Setenv("REDIS_TLS_URL", "rediss://:p@secure-redis.example.com:6380/0")

	got := getRedisURLFromEnv()
	want := "rediss://:p@secure-redis.example.com:6380/0"
	if got != want {
		t.Fatalf("getRedisURLFromEnv() = %q, want %q", got, want)
	}
}
