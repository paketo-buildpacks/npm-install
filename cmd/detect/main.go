package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libcfbuildpack/detect"
	"github.com/cloudfoundry/npm-cnb/detection"
)

func main() {
	context, err := detect.DefaultDetect()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create default detect context: %s", err)
		os.Exit(100)
	}

	version, err := detection.GetNodeVersion(filepath.Join(context.Application.Root, "package.json"))
	if err != nil {
		if !os.IsNotExist(err) {
			context.Logger.Info(err.Error())
		}

		os.Exit(detect.FailStatusCode)
	}

	code, err := context.Pass(detection.NewPlan(version))
	if err != nil {
		context.Logger.Info(err.Error())
	}

	os.Exit(code)
}
