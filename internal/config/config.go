package config

import (
	"fmt"
	"os"
	"strings"
)

const httpAddressEnv = "HTTP_ADDR"

type Config struct {
	HTTPAddress string
}

func Load() (Config, error) {
	httpAddress := strings.TrimSpace(os.Getenv(httpAddressEnv))
	if httpAddress == "" {
		return Config{}, fmt.Errorf("%s is required", httpAddressEnv)
	}

	return Config{HTTPAddress: httpAddress}, nil
}
