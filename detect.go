package npminstall

import (
	"errors"
	"os"
	"path/filepath"

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

//go:generate faux --interface PathParser --output fakes/path_parser.go
type PathParser interface {
	Get(path string) (projectPath string, err error)
}

func Detect(projectPathParser PathParser, packageJSONParser VersionParser) packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {

		projectPath, err := projectPathParser.Get(context.WorkingDir)
		if err != nil {
			return packit.DetectResult{}, err
		}

		version, err := packageJSONParser.ParseVersion(filepath.Join(context.WorkingDir, projectPath, "package.json"))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return packit.DetectResult{}, packit.Fail.WithMessage(`no "package.json" found in project path %s`, filepath.Join(context.WorkingDir, projectPath))
			}

			return packit.DetectResult{}, err
		}

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

		return packit.DetectResult{
			Plan: packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{Name: NodeModules},
				},
				Requires: []packit.BuildPlanRequirement{
					nodeDependency,
					{
						Name: Npm,
						Metadata: BuildPlanMetadata{
							Build: true,
						},
					},
				},
			},
		}, nil
	}
}
