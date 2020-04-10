package main

import (
	"os"
	"time"

	"github.com/cloudfoundry/packit"
	"github.com/cloudfoundry/packit/fs"
	"github.com/cloudfoundry/packit/pexec"
	"github.com/cloudfoundry/packit/scribe"
	"github.com/paketo-buildpacks/npm/npm"
)

func main() {
	executable := pexec.NewExecutable("npm")
	logger := scribe.NewLogger(os.Stdout)

	checksumCalculator := fs.NewChecksumCalculator()
	resolver := npm.NewBuildProcessResolver(executable, checksumCalculator, &logger)
	clock := npm.NewClock(time.Now)

	packit.Build(npm.Build(resolver, clock, &logger))
}
