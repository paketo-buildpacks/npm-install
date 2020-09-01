package npm

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/packit"
)

const (
	PlanDependencyNodeModules = "node_modules"
	PlanDependencyNode        = "node"
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

func Detect(packageJSONParser VersionParser) packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		version, err := packageJSONParser.ParseVersion(filepath.Join(context.WorkingDir, "package.json"))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return packit.DetectResult{}, packit.Fail
			}

			return packit.DetectResult{}, err
		}

		nodeDependency := packit.BuildPlanRequirement{
			Name: PlanDependencyNode,
			Metadata: BuildPlanMetadata{
				Build:  true,
				Launch: true,
			},
		}

		if version != "" {
			nodeDependency.Metadata = BuildPlanMetadata{
				Version:       version,
				VersionSource: "package.json",
				Build:         true,
				Launch:        true,
			}
		}

		return packit.DetectResult{
			Plan: packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{Name: PlanDependencyNodeModules},
				},
				Requires: []packit.BuildPlanRequirement{
					{Name: PlanDependencyNodeModules},
					nodeDependency,
				},
			},
		}, nil
	}
}
