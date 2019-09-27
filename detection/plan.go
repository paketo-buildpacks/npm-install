package detection

import (
	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/npm-cnb/modules"
)

func NewPlan(version string) buildplan.Plan {
	var source string
	if version != "" {
		source = "package.json"
	}

	return buildplan.Plan{
		Provides: []buildplan.Provided{
			{Name: modules.Dependency},
		},
		Requires: []buildplan.Required{
			{
				Name:    modules.NodeDependency,
				Version: version,
				Metadata: buildplan.Metadata{
					"build":          true,
					"launch":         true,
					"version-source": source,
				},
			},
			{
				Name: modules.Dependency,
				Metadata: buildplan.Metadata{
					"launch": true,
				},
			},
		},
	}
}
