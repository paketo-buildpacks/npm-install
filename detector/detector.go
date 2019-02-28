package detector

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/detect"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/npm-cnb/modules"
)

type Detector struct{}

func (d *Detector) RunDetect(context detect.Detect) (int, error) {
	packageJSON := filepath.Join(context.Application.Root, "package.json")
	packageJSONVersion, err := d.checkPackageJSON(packageJSON)
	if err != nil {
		return detect.FailStatusCode, err
	}

	nodePlan, err := d.nodeBuildPlan(context, packageJSONVersion)
	if err != nil {
		return detect.FailStatusCode, err
	}

	return context.Pass(buildplan.BuildPlan{
		modules.NodeDependency: nodePlan,
		modules.Dependency:     buildplan.Dependency{Metadata: buildplan.Metadata{"launch": true}},
	})
}

func (d *Detector) nodeBuildPlan(context detect.Detect, packageJSONVersion string) (buildplan.Dependency, error) {
	nodePlan := buildplan.Dependency{Metadata: buildplan.Metadata{"build": true, "launch": true}}

	nodePlan.Version = d.getBuildPlanVersion(context)

	if exists, err := helper.FileExists(filepath.Join(context.Application.Root, ".nvmrc")); err != nil {
		return buildplan.Dependency{}, err
	} else if exists {
		warnNodeEngine(nodePlan.Version, packageJSONVersion, context)
	}

	if packageJSONVersion != "" {
		nodePlan.Version = packageJSONVersion
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

func (d *Detector) getBuildPlanVersion(context detect.Detect) string {
	if nodeDep, found := context.BuildPlan[modules.NodeDependency]; found {
		return nodeDep.Version
	}
	return ""
}

func warnNodeEngine(nvmrcNodeVersion string, packageJSONNodeVersion string, context detect.Detect) []string {
	docsLink := "http://docs.cloudfoundry.org/buildpacks/node/node-tips.html"

	var logs []string
	if nvmrcNodeVersion != "" && packageJSONNodeVersion == "" {
		context.Logger.Info("Using the node version specified in your .nvmrc See: %s", docsLink)
	}
	if packageJSONNodeVersion != "" && nvmrcNodeVersion != "" {
		context.Logger.Info("Node version in .nvmrc ignored in favor of 'engines' field in package.json")
	}
	if packageJSONNodeVersion == "" && nvmrcNodeVersion == "" {
		context.Logger.Info("Node version not specified in package.json or .nvmrc. See: %s", docsLink)
	}
	if packageJSONNodeVersion == "*" {
		context.Logger.Info("Dangerous semver range (*) in engines.node. See: %s", docsLink)
	}
	if strings.HasPrefix(packageJSONNodeVersion, ">") {
		context.Logger.Info("Dangerous semver range (>) in engines.node. See: %s", docsLink)
	}
	return logs
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
