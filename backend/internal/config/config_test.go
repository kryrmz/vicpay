package config

import "testing"

func devConfig() *Config {
	c := Load()
	c.Environment = EnvDevelopment
	return c
}

func TestValidateForProductionAllowsDev(t *testing.T) {
	if err := devConfig().ValidateForProduction(); err != nil {
		t.Fatalf("development must always pass: %v", err)
	}
}

func TestValidateForProductionRejectsDevDefaults(t *testing.T) {
	c := Load()
	c.Environment = EnvProduction
	if err := c.ValidateForProduction(); err == nil {
		t.Fatal("production with dev defaults must fail")
	}
}

func TestValidateForProductionPassesWhenHardened(t *testing.T) {
	c := hardenedProd()
	if err := c.ValidateForProduction(); err != nil {
		t.Fatalf("hardened production config should pass: %v", err)
	}
}

func TestValidateForProductionRejectsNonTLSDBByDefault(t *testing.T) {
	c := hardenedProd()
	c.DatabaseURL = "postgres://app:pw@pgbouncer:6432/vicpay?sslmode=disable"
	c.DBTrustedNetwork = false
	if err := c.ValidateForProduction(); err == nil {
		t.Fatal("non-TLS DATABASE_URL must fail without DB_TRUSTED_NETWORK")
	}
}

func TestValidateForProductionAllowsNonTLSDBOnTrustedNetwork(t *testing.T) {
	c := hardenedProd()
	c.DatabaseURL = "postgres://app:pw@pgbouncer:6432/vicpay?sslmode=disable"
	c.DBTrustedNetwork = true
	if err := c.ValidateForProduction(); err != nil {
		t.Fatalf("non-TLS DATABASE_URL should pass on a trusted network: %v", err)
	}
}

// hardenedProd returns a production config with all insecure defaults replaced.
func hardenedProd() *Config {
	c := Load()
	c.Environment = EnvProduction
	c.JWTSecret = "a-strong-production-secret-value-32bytes!"
	c.PIIEncryptKey = "another-strong-32-byte-production-key!!!"
	c.DatabaseURL = "postgres://vicpay:realpw@db.internal:6432/vicpay?sslmode=require"
	c.CORSOrigins = []string{"https://app.vicpay.example"}
	c.OTPDevEcho = false
	return c
}
