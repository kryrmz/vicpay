// Package config loads runtime configuration from the environment. Dev defaults
// are intentionally insecure but ValidateForProduction refuses to boot outside
// development if any of them survive. Two database URLs are kept distinct:
// DATABASE_URL (may route through a transactional pooler like PgBouncer) and
// DATABASE_DIRECT_URL (a direct connection required for migrations and session
// advisory locks, which break under transaction pooling).
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Environment names.
const (
	EnvDevelopment = "development"
	EnvStaging     = "staging"
	EnvProduction  = "production"
)

// Config is the fully resolved application configuration.
type Config struct {
	Environment string
	Port        int

	DatabaseURL       string // app pool; may go through PgBouncer (transaction mode)
	DatabaseDirectURL string // direct to Postgres; migrations + advisory locks

	RedisURL string // optional; empty enables the in-memory fallback

	JWTSecret     string
	JWTIssuer     string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
	IdleTimeout   time.Duration
	PIIEncryptKey string // 32-byte key for app-layer AES-256-GCM PII encryption

	CORSOrigins []string

	// OTPDevEcho, only honored in development, logs the OTP code to the server
	// log so a developer can complete the flow without a real SMS provider.
	// It is refused by ValidateForProduction.
	OTPDevEcho bool

	// MigrateOnly runs the migrations and exits, without starting the server.
	// Used to apply the schema (as the owner) before the least-privilege app
	// role exists, in a single-host deployment.
	MigrateOnly bool

	// DBTrustedNetwork asserts that DATABASE_URL travels only over a trusted
	// private network (e.g. a single-host Docker bridge where Postgres is not
	// publicly exposed). It is the explicit, documented opt-in that lets
	// ValidateForProduction accept a non-TLS app DSN in that topology. TLS to a
	// remote/managed database is still required when this is false.
	DBTrustedNetwork bool
}

// Load reads configuration from the environment, applying dev defaults.
func Load() *Config {
	c := &Config{
		Environment:       getEnv("ENVIRONMENT", EnvDevelopment),
		Port:              getEnvInt("PORT", 8080),
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://vicpay:vicpay_dev@localhost:6432/vicpay?sslmode=disable"),
		DatabaseDirectURL: getEnv("DATABASE_DIRECT_URL", "postgres://vicpay:vicpay_dev@localhost:5432/vicpay?sslmode=disable"),
		RedisURL:          getEnv("REDIS_URL", ""),
		JWTSecret:         getEnv("JWT_SECRET", "dev-secret-change-in-production-000"),
		JWTIssuer:         getEnv("JWT_ISSUER", "vicpay"),
		AccessTTL:         time.Duration(getEnvInt("JWT_ACCESS_MINUTES", 15)) * time.Minute,
		RefreshTTL:        time.Duration(getEnvInt("JWT_REFRESH_DAYS", 7)) * 24 * time.Hour,
		IdleTimeout:       time.Duration(getEnvInt("JWT_IDLE_MINUTES", 30)) * time.Minute,
		PIIEncryptKey:     getEnv("PII_ENCRYPTION_KEY", "dev-pii-key-change-me-32bytes-min!"),
		CORSOrigins:       splitCSV(getEnv("CORS_ORIGINS", "http://localhost:5173,http://localhost:9999")),
		OTPDevEcho:        getEnvBool("OTP_DEV_ECHO", true),
		MigrateOnly:       getEnvBool("MIGRATE_ONLY", false),
		DBTrustedNetwork:  getEnvBool("DB_TRUSTED_NETWORK", false),
	}
	return c
}

// IsDevelopment reports whether the app runs in the development environment.
func (c *Config) IsDevelopment() bool { return c.Environment == EnvDevelopment }

// ValidateForProduction fails closed: outside development it rejects any insecure
// default that survived. This is the cheapest guard against shipping dev secrets.
func (c *Config) ValidateForProduction() error {
	if c.IsDevelopment() {
		return nil
	}
	var problems []string
	if strings.HasPrefix(c.JWTSecret, "dev-secret") || len(c.JWTSecret) < 32 {
		problems = append(problems, "JWT_SECRET must be set to a strong value (>=32 bytes)")
	}
	if strings.HasPrefix(c.PIIEncryptKey, "dev-pii-key") || len(c.PIIEncryptKey) < 32 {
		problems = append(problems, "PII_ENCRYPTION_KEY must be set to a strong value (>=32 bytes)")
	}
	if strings.Contains(c.DatabaseURL, "sslmode=disable") && !c.DBTrustedNetwork {
		problems = append(problems, "DATABASE_URL must use TLS (sslmode!=disable), or set DB_TRUSTED_NETWORK=true for a private single-host network")
	}
	if strings.Contains(c.DatabaseURL, "vicpay_dev") {
		problems = append(problems, "DATABASE_URL still uses the dev password")
	}
	for _, o := range c.CORSOrigins {
		if o == "*" {
			problems = append(problems, "CORS_ORIGINS must not be a wildcard")
		}
	}
	if c.OTPDevEcho {
		problems = append(problems, "OTP_DEV_ECHO must be false outside development")
	}
	if len(problems) > 0 {
		return fmt.Errorf("insecure production config: %s", strings.Join(problems, "; "))
	}
	return nil
}

// PIIKey returns the PII encryption key as bytes, erroring if it is too short.
func (c *Config) PIIKey() ([]byte, error) {
	if len(c.PIIEncryptKey) < 32 {
		return nil, errors.New("config: PII_ENCRYPTION_KEY must be at least 32 bytes")
	}
	return []byte(c.PIIEncryptKey)[:32], nil
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	if v, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
