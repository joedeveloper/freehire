package sources

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the parsed sources.yml: the set of boards to crawl.
type Config struct {
	Sources []CompanyEntry `yaml:"sources"`
}

// LoadConfig reads and parses a sources.yml file.
func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("sources: read config %s: %w", path, err)
	}
	return ParseConfig(data)
}

// ParseConfig parses sources.yml bytes into a Config.
func ParseConfig(data []byte) (Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("sources: parse config: %w", err)
	}
	return cfg, nil
}

// Validate checks every configured entry names a registered provider, so the ingest
// command can fail fast instead of silently skipping a misconfigured board.
func (c Config) Validate(registry map[string]Source) error {
	for _, e := range c.Sources {
		if e.Company == "" {
			return fmt.Errorf("sources: entry with provider %q has empty company", e.Provider)
		}
		if e.Board == "" {
			return fmt.Errorf("sources: entry for company %q has empty board", e.Company)
		}
		if _, ok := registry[e.Provider]; !ok {
			return fmt.Errorf("sources: unknown provider %q for company %q", e.Provider, e.Company)
		}
	}
	return nil
}
