package npminstall

import (
	"os"
)

type Environment struct {
	defaultValues map[string]string
}

func NewEnvironment() Environment {
	return Environment{
		defaultValues: map[string]string{
			"NPM_CONFIG_LOGLEVEL": "error",
		},
	}
}

func (e Environment) GetValue(key string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return e.defaultValues[key]
}
