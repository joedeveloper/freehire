package sources

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is a parsed board file: the boards to crawl plus the file's default provider
// (its base name). Each entry's provider is normally this default, but an entry may name
// its own, so one file can list boards for several providers (e.g. a shared custom.yml).
type Config struct {
	Provider string
	Sources  []CompanyEntry
}

// LoadConfig reads a board file (e.g. sources/greenhouse.yml or sources/custom.yml). The
// file's base name is the default provider; an entry that names its own provider keeps it,
// so a per-provider file repeats nothing while a mixed file names the provider per entry.
func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("sources: read config %s: %w", path, err)
	}
	provider := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return ParseConfig(provider, data)
}

// ParseConfig parses a board-list, filling the file's default provider only where an entry
// left it blank — an entry's own provider wins — so every CompanyEntry ends up with a
// provider set for the rest of the pipeline.
func ParseConfig(provider string, data []byte) (Config, error) {
	var entries []CompanyEntry
	if err := yaml.Unmarshal(data, &entries); err != nil {
		return Config{}, fmt.Errorf("sources: parse config: %w", err)
	}
	for i := range entries {
		if entries[i].Provider == "" {
			entries[i].Provider = provider
		}
	}
	return Config{Provider: provider, Sources: entries}, nil
}

// Validate checks every entry against the registry by its own resolved provider, so the
// ingest command fails fast instead of silently skipping a misconfigured board. Each
// entry's provider is its own when set, else the file's default.
func (c Config) Validate(registry map[string]Source) error {
	for _, e := range c.Sources {
		provider := e.Provider
		if provider == "" {
			provider = c.Provider
		}
		src, ok := registry[provider]
		if !ok {
			return fmt.Errorf("sources: unknown provider %q", provider)
		}
		if e.Company == "" {
			return fmt.Errorf("sources: %s entry has empty company", provider)
		}
		// A boardless provider crawls one company's own API and has no board id, so its
		// entries may omit board; every other provider still requires one.
		if _, noBoard := src.(boardless); !noBoard && e.Board == "" {
			return fmt.Errorf("sources: %s entry for company %q has empty board", provider, e.Company)
		}
	}
	return nil
}
