package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/libcfbuildpack/helper"

	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/detect"
	"github.com/cloudfoundry/nodejs-cnb/node"
	"github.com/cloudfoundry/npm-cnb/modules"
)

func main() {
	context, err := detect.DefaultDetect()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to create default detect context: %s", err)
		os.Exit(100)
	}

	if err := context.BuildPlan.Init(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to initialize Build Plan: %s\n", err)
		os.Exit(101)
	}

	code, err := runDetect(context)
	if err != nil {
		context.Logger.Info(err.Error())
	}

	os.Exit(code)
}

func runDetect(context detect.Detect) (int, error) {
	packageJSON := filepath.Join(context.Application.Root, "package.json")
	if exists, err := helper.FileExists(packageJSON); err != nil {
		return context.Fail(), fmt.Errorf("error checking filepath: %s", packageJSON)
	} else if !exists {
		return context.Fail(), fmt.Errorf(`no "package.json" found at: %s`, packageJSON)
	}

	packageJSONVersion, err := node.GetVersion(packageJSON)
	if err != nil {
		return context.Fail(), fmt.Errorf(`unable to parse "package.json": %s`, err.Error())
	}

	curNodeVersion := ""
	if nodeDep, found := context.BuildPlan[node.Dependency]; found {
		curNodeVersion = nodeDep.Version
	}

	if exists, err := helper.FileExists(filepath.Join(context.Application.Root, ".nvmrc")); err != nil {
		return context.Fail(), err
	} else if exists {
		warnNodeEngine(curNodeVersion, packageJSONVersion, context)
	}

	version := ""
	if packageJSONVersion != "" {
		version = packageJSONVersion
	} else if curNodeVersion != "" {
		version = curNodeVersion
	}

	return context.Pass(buildplan.BuildPlan{
		node.Dependency: buildplan.Dependency{
			Version:  version,
			Metadata: buildplan.Metadata{"build": true, "launch": true},
		},
		modules.Dependency: buildplan.Dependency{
			Metadata: buildplan.Metadata{"launch": true},
		},
	})
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
