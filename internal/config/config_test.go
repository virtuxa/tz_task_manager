package config

import "testing"

func TestLoad(t *testing.T) {
	t.Setenv(httpAddressEnv, ":8080")
	t.Setenv(mysqlDSNEnv, "user:password@tcp(localhost:3306)/task_manager")
	t.Setenv(redisAddressEnv, "localhost:6379")
	t.Setenv(jwtSecretEnv, "test-signing-key-with-at-least-thirty-two-bytes")
	t.Setenv(jwtTTLEnv, "24h")

	config, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if config.HTTPAddress != ":8080" {
		t.Fatalf("HTTP address = %q, want %q", config.HTTPAddress, ":8080")
	}

	if config.MySQLDSN != "user:password@tcp(localhost:3306)/task_manager" {
		t.Fatalf("MySQL DSN = %q", config.MySQLDSN)
	}

	if config.RedisAddress != "localhost:6379" {
		t.Fatalf("Redis address = %q", config.RedisAddress)
	}

	if config.JWTSecret != "test-signing-key-with-at-least-thirty-two-bytes" {
		t.Fatalf("JWT secret = %q", config.JWTSecret)
	}

	if config.JWTTTL.String() != "24h0m0s" {
		t.Fatalf("JWT TTL = %s", config.JWTTTL)
	}
}

func TestLoadRequiresMySQLDSN(t *testing.T) {
	t.Setenv(httpAddressEnv, ":8080")
	t.Setenv(mysqlDSNEnv, "")
	t.Setenv(redisAddressEnv, "localhost:6379")
	t.Setenv(jwtSecretEnv, "test-signing-key-with-at-least-thirty-two-bytes")
	t.Setenv(jwtTTLEnv, "24h")

	if _, err := Load(); err == nil {
		t.Fatal("load config must fail without MySQL DSN")
	}
}
