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
	c := Load()
	c.Environment = EnvProduction
	c.JWTSecret = "a-strong-production-secret-value-32bytes!"
	c.PIIEncryptKey = "another-strong-32-byte-production-key!!!"
	c.DatabaseURL = "postgres://vicpay:realpw@db.internal:6432/vicpay?sslmode=require"
	c.CORSOrigins = []string{"https://app.vicpay.example"}
	c.OTPDevEcho = false
	if err := c.ValidateForProduction(); err != nil {
		t.Fatalf("hardened production config should pass: %v", err)
	}
}
