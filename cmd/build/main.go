package main

import (
	"fmt"
	"os"
	"time"

	"github.com/cloudfoundry/libcfbuildpack/build"
	"github.com/cloudfoundry/npm-cnb/modules"
	"github.com/cloudfoundry/npm-cnb/npm"
)

func main() {
	context, err := build.DefaultBuild()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to create default build context: %s", err)
		os.Exit(100)
	}

	context.Logger.FirstLine(context.Logger.PrettyIdentity(context.Buildpack))

	contributor, willContribute, err := modules.NewContributor(context, npm.NPM{
		Runner: npm.Command{},
		Logger: context.Logger,
	})
	if err != nil {
		context.Logger.Info(err.Error())
		os.Exit(context.Failure(102))
	}

	if willContribute {
		if err := contributor.Contribute(time.Now()); err != nil {
			context.Logger.Info(err.Error())
			os.Exit(context.Failure(103))
		}
	}

	code, err := context.Success()
	if err != nil {
		context.Logger.Info(err.Error())
	}

	os.Exit(code)
}
