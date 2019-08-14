package detector

import (
	"encoding/json"
	"fmt"
	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/detect"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/npm-cnb/modules"
	"os"
	"path/filepath"
)

type Detector struct{}

func (d *Detector) RunDetect(context detect.Detect) (int, error) {
	packageJSON := filepath.Join(context.Application.Root, "package.json")
	if exists, err := helper.FileExists(packageJSON); err != nil {
		return detect.FailStatusCode, err
	} else if !exists {
		return detect.FailStatusCode, nil
	}

	packageJSONVersion, err := GetVersion(packageJSON)
	if err != nil {
		return detect.FailStatusCode, fmt.Errorf(`unable to parse "package.json": %s`, err.Error())
	}

	nodePlan, err := d.nodeBuildPlan(context, packageJSONVersion)
	if err != nil {
		return detect.FailStatusCode, err
	}

	return context.Pass(buildplan.Plan{
		Provides: []buildplan.Provided{{Name: modules.Dependency}},
		Requires: []buildplan.Required{
			nodePlan,
			{
				Name:     modules.Dependency,
				Metadata: buildplan.Metadata{"launch": true},
			},
		},
	})
}

func (d *Detector) nodeBuildPlan(context detect.Detect, packageJSONVersion string) (buildplan.Required, error) {
	versionSource := ""
	if packageJSONVersion != "" {
		versionSource = "package.json"
	}

	nodePlan := buildplan.Required{
		Name:     modules.NodeDependency,
		Version:  packageJSONVersion,
		Metadata: buildplan.Metadata{"build": true, "launch": true, "version-source": versionSource},
	}

	return nodePlan, nil
}

func (d *Detector) checkPackageJSON(packageJSON string) (string, error) {
	if exists, err := helper.FileExists(packageJSON); err != nil {
		return "", fmt.Errorf("error checking filepath: %s", packageJSON)
	} else if !exists {
		return "", fmt.Errorf(`no "package.json" found at: %s`, packageJSON)
	}

	packageJSONVersion, err := GetVersion(packageJSON)
	if err != nil {
		return "", fmt.Errorf(`unable to parse "package.json": %s`, err.Error())
	}

	return packageJSONVersion, nil
}

type packageJSON struct {
	Engines engines `json:"engines"`
}

type engines struct {
	Node string `json:"node"`
}

func GetVersion(packageFile string) (version string, err error) {
	file, err := os.Open(packageFile)
	if err != nil {
		return "", err
	}
	defer file.Close()

	pkgJSON := packageJSON{}
	if err := json.NewDecoder(file).Decode(&pkgJSON); err != nil {
		return "", err
	}

	return pkgJSON.Engines.Node, nil
}
