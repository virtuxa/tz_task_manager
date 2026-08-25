package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	httpAddressEnv = "HTTP_ADDR"
	mysqlDSNEnv    = "MYSQL_DSN"
	jwtSecretEnv   = "JWT_SECRET"
	jwtTTLEnv      = "JWT_TTL"
)

type Config struct {
	HTTPAddress string
	MySQLDSN    string
	JWTSecret   string
	JWTTTL      time.Duration
}

func Load() (Config, error) {
	httpAddress, err := requiredEnv(httpAddressEnv)
	if err != nil {
		return Config{}, err
	}

	mysqlDSN, err := requiredEnv(mysqlDSNEnv)
	if err != nil {
		return Config{}, err
	}

	jwtSecret, err := requiredEnv(jwtSecretEnv)
	if err != nil {
		return Config{}, err
	}

	jwtTTLRaw, err := requiredEnv(jwtTTLEnv)
	if err != nil {
		return Config{}, err
	}

	jwtTTL, err := time.ParseDuration(jwtTTLRaw)
	if err != nil || jwtTTL <= 0 {
		return Config{}, fmt.Errorf("%s must be a positive duration", jwtTTLEnv)
	}

	return Config{
		HTTPAddress: httpAddress,
		MySQLDSN:    mysqlDSN,
		JWTSecret:   jwtSecret,
		JWTTTL:      jwtTTL,
	}, nil
}

func requiredEnv(name string) (string, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return "", fmt.Errorf("%s is required", name)
	}

	return value, nil
}
