package npminstall

import (
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/scribe"
)

type Environment struct {
	logger        scribe.Logger
	defaultValues map[string]string
}

func NewEnvironment(logger scribe.Logger) Environment {
	return Environment{
		logger: logger,
		defaultValues: map[string]string{
			"NPM_CONFIG_LOGLEVEL": "error",
		},
	}
}

func (e Environment) Configure(layer packit.Layer) error {
	for envvar := range e.defaultValues {
		layer.LaunchEnv.Default(envvar, e.GetValue(envvar))
	}

	path := filepath.Join(layer.Path, "node_modules", ".bin")
	layer.SharedEnv.Append("PATH", path, string(os.PathListSeparator))

	e.logger.Process("Configuring environment")
	e.logger.Subprocess("%s", scribe.NewFormattedMapFromEnvironment(layer.LaunchEnv))
	e.logger.Subprocess("%s", scribe.NewFormattedMapFromEnvironment(layer.SharedEnv))

	return nil
}

func (e Environment) GetValue(key string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return e.defaultValues[key]
}
