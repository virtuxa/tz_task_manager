package config

import "testing"

func TestLoad(t *testing.T) {
	t.Setenv(httpAddressEnv, ":8080")
	t.Setenv(mysqlDSNEnv, "user:password@tcp(localhost:3306)/task_manager")

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
}

func TestLoadRequiresMySQLDSN(t *testing.T) {
	t.Setenv(httpAddressEnv, ":8080")
	t.Setenv(mysqlDSNEnv, "")

	if _, err := Load(); err == nil {
		t.Fatal("load config must fail without MySQL DSN")
	}
}
