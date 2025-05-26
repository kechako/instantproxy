package config

import (
	"fmt"
	"net/url"
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Servers []*Server `toml:"servers"`
}

func (cfg *Config) genProxyTable() (map[string]*url.URL, error) {
	table := map[string]*url.URL{}
	for _, server := range cfg.Servers {
		backendURL, err := url.Parse(server.BackendURL)
		if err != nil {
			return nil, fmt.Errorf("backend_url is not valid: %s: %w", server.BackendURL, err)
		}
		table[server.Host] = backendURL
	}

	return table, nil
}

type Server struct {
	Host       string `toml:"host"`
	BackendURL string `toml:"backend_url"`
}

func Load(name string) (*Config, error) {
	b, err := os.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %s: %w", name, err)
	}

	cfg := &Config{}
	err = toml.Unmarshal(b, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %s: %w", name, err)
	}

	return cfg, nil
}
