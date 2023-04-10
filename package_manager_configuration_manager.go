package npminstall

import (
	"fmt"
	"path/filepath"

	"github.com/paketo-buildpacks/packit/v2/scribe"
	"github.com/paketo-buildpacks/packit/v2/servicebindings"
)

//go:generate faux --interface BindingResolver --output fakes/binding_resolver.go
type BindingResolver interface {
	Resolve(typ, provider, platformDir string) ([]servicebindings.Binding, error)
}

type PackageManagerConfigurationManager struct {
	bindingResolver BindingResolver
	logs            scribe.Emitter
	defaultPath     string
}

func NewPackageManagerConfigurationManager(bindingResolver BindingResolver, logs scribe.Emitter, defaultPath string) PackageManagerConfigurationManager {
	return PackageManagerConfigurationManager{
		bindingResolver: bindingResolver,
		logs:            logs,
		defaultPath:     defaultPath,
	}
}

func (p PackageManagerConfigurationManager) DeterminePath(typ, platformDir, entry string) (string, error) {
	if p.defaultPath != "" {
		return p.defaultPath, nil
	}

	bindings, err := p.bindingResolver.Resolve(typ, "", platformDir)
	if err != nil {
		return "", err
	}

	if len(bindings) > 1 {
		return "", fmt.Errorf("failed: binding resolver found more than one binding of type '%s'", typ)
	}

	if len(bindings) == 1 {
		p.logs.Process("Loading service binding of type '%s'", typ)

		if _, ok := bindings[0].Entries[entry]; !ok {
			return "", fmt.Errorf("failed: binding of type '%s' does not contain required entry '%s'", typ, entry)
		}

		return filepath.Join(bindings[0].Path, entry), nil
	}

	return "", nil
}
