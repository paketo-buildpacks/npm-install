package main

import (
	"fmt"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"os"
	"path/filepath"

	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/detect"
	"github.com/cloudfoundry/npm-cnb/modules"
	"github.com/cloudfoundry/npm-cnb/node"
)

func main() {
	context, err := detect.DefaultDetect()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to create default detect context: %s", err)
		os.Exit(100)
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

	version, err := node.GetNodeVersion(packageJSON)
	if err != nil {
		return context.Fail(), fmt.Errorf(`unable to parse "package.json": %s`, err.Error())
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
