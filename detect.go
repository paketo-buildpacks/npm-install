package npminstall

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/libnodejs"
	"github.com/paketo-buildpacks/packit/v2"
)

type BuildPlanMetadata struct {
	Version       string `toml:"version"`
	VersionSource string `toml:"version-source"`
	Build         bool   `toml:"build"`
	Launch        bool   `toml:"launch"`
}

//go:generate faux --interface VersionParser --output fakes/version_parser.go
type VersionParser interface {
	ParseVersion(path string) (version string, err error)
}

func Detect() packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {

		projectPath, err := libnodejs.FindProjectPath(context.WorkingDir)
		if err != nil {
			return packit.DetectResult{}, err
		}

		pkg, err := libnodejs.ParsePackageJSON(projectPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return packit.DetectResult{}, packit.Fail.WithMessage("no 'package.json' found in project path %s", filepath.Join(projectPath))
			}

			return packit.DetectResult{}, err
		}
		version := pkg.GetVersion()

		_, nodeGypInDep := pkg.Dependencies["node-gyp"]
		_, nodeGypInDevDep := pkg.DevDependencies["node-gyp"]
		pythonNeeded := nodeGypInDep || nodeGypInDevDep

		nodeDependency := packit.BuildPlanRequirement{
			Name: Node,
			Metadata: BuildPlanMetadata{
				Build: true,
			},
		}

		if version != "" {
			nodeDependency.Metadata = BuildPlanMetadata{
				Version:       version,
				VersionSource: "package.json",
				Build:         true,
			}
		}

		npmDependency := packit.BuildPlanRequirement{
			Name: Npm,
			Metadata: BuildPlanMetadata{
				Build: true,
			},
		}

		cPythonDependency := packit.BuildPlanRequirement{
			Name: Cpython,
			Metadata: BuildPlanMetadata{
				Build: true,
			},
		}

		result := packit.DetectResult{
			Plan: packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{Name: NodeModules},
				},
				Requires: []packit.BuildPlanRequirement{
					nodeDependency,
					npmDependency,
				},
			},
		}

		if pythonNeeded {
			result.Plan.Requires = append(result.Plan.Requires, cPythonDependency)
		}

		return result, nil
	}
}
