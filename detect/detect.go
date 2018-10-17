package detect

import (
	"fmt"
	"path/filepath"

	libbuildpackV3 "github.com/buildpack/libbuildpack"
	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/npm-cnb/package_json"
)

const NodeDependency = "node"
const ModulesDependency = "modules"

func UpdateBuildPlan(libDetect *libbuildpackV3.Detect) error {
	packageJSONPath := filepath.Join(libDetect.Application.Root, "package.json")
	if exists, err := libbuildpack.FileExists(packageJSONPath); err != nil {
		return fmt.Errorf("error checking filepath %s", packageJSONPath)
	} else if !exists {
		return fmt.Errorf("no package.json found in %s", packageJSONPath)
	}

	pkgJSON, err := package_json.LoadPackageJSON(packageJSONPath, libDetect.Logger)
	if err != nil {
		return err
	}

	libDetect.BuildPlan[NodeDependency] = libbuildpackV3.BuildPlanDependency{
		Version: pkgJSON.Engines.Node,
		Metadata: libbuildpackV3.BuildPlanDependencyMetadata{
			"build":  true,
			"launch": true,
		},
	}

	libDetect.BuildPlan[ModulesDependency] = libbuildpackV3.BuildPlanDependency{
		Metadata: libbuildpackV3.BuildPlanDependencyMetadata{
			"launch": true,
		},
	}

	return nil
}
