package main

import (
	"fmt"
	"os"
	
	"github.com/buildpack/libbuildpack"
	"github.com/cloudfoundry/npm-cnb/internal/detect"
)

func main() {
	detector, err := libbuildpack.DefaultDetect()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create default detector: %s", err)
		os.Exit(100)
	}

	if err := detect.UpdateBuildPlan(&detector); err != nil {
		detector.Logger.Debug("failed npm detection: %s", err)
		detector.Fail()
	}

	detector.Pass(detector.BuildPlan)
}
