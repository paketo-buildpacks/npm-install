package main

import (
	"fmt"
	"os"

	"github.com/cloudfoundry/libjavabuildpack"
	"github.com/cloudfoundry/npm-cnb/build"
	"github.com/cloudfoundry/npm-cnb/npm"
)

func main() {
	builder, err := libjavabuildpack.DefaultBuild()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create default builder: %s", err)
		os.Exit(100)
	}

	modules, ok, err := build.NewModules(builder, &npm.NPM{})

	if err != nil {
		builder.Logger.Info(err.Error())
		builder.Failure(102)
		return
	}

	if ok {
		if err := modules.Contribute(); err != nil {
			builder.Logger.Info(err.Error())
			builder.Failure(103)
			return
		}
	}

	if err := builder.Launch.WriteMetadata(modules.CreateLaunchMetadata()); err != nil {
		builder.Logger.Info("failed to write launch.toml: %s", err)
		builder.Failure(100)
	}

	builder.Success()
}
