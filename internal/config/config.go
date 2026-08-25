package config

import (
	"fmt"
	"os"
	"strings"
)

const (
	httpAddressEnv = "HTTP_ADDR"
	mysqlDSNEnv    = "MYSQL_DSN"
)

type Config struct {
	HTTPAddress string
	MySQLDSN    string
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

	return Config{
		HTTPAddress: httpAddress,
		MySQLDSN:    mysqlDSN,
	}, nil
}

func requiredEnv(name string) (string, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return "", fmt.Errorf("%s is required", name)
	}

	return value, nil
}
