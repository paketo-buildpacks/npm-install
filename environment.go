package npminstall

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

type Environment struct {
	store map[string]string
}

func ParseEnvironment(path string, variables []string) (Environment, error) {
	file, err := os.Open(path)
	if err != nil {
		return Environment{}, fmt.Errorf("failed to read \"buildpack.toml\": %w", err)
	}
	defer file.Close()

	var configuration struct {
		Metadata struct {
			Configurations []struct {
				Default     string `toml:"default,omitempty"`
				Description string `toml:"description"`
				Name        string `toml:"name"`
			} `toml:"configurations"`
		} `toml:"metadata"`
	}
	_, err = toml.NewDecoder(file).Decode(&configuration)
	if err != nil {
		return Environment{}, fmt.Errorf("failed to parse \"buildpack.toml\": %w", err)
	}

	store := make(map[string]string)
	for _, configuration := range configuration.Metadata.Configurations {
		store[configuration.Name] = configuration.Default
	}

	environ := make(map[string]string)
	for _, variable := range variables {
		if key, value, found := strings.Cut(variable, "="); found {
			environ[key] = value
		}
	}

	for key, def := range store {
		if value, ok := environ[key]; ok {
			store[key] = value
		} else {
			if def == "" {
				delete(store, key)
			}
		}
	}

	return Environment{store: store}, nil
}

func (e Environment) Lookup(key string) (string, bool) {
	value, found := e.store[key]
	return value, found
}

func (e Environment) LookupBool(key string) (bool, error) {
	if value, found := e.Lookup(key); found {
		result, err := strconv.ParseBool(value)
		if err != nil {
			return false, fmt.Errorf("failed to parse boolean environment variable %q: %w", key, err)
		}

		return result, nil
	}

	return false, nil
}
