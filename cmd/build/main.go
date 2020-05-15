package main

import (
	"os"
	"time"

	"github.com/paketo-buildpacks/npm/npm"
	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/fs"
	"github.com/paketo-buildpacks/packit/pexec"
	"github.com/paketo-buildpacks/packit/scribe"
)

func main() {
	executable := pexec.NewExecutable("npm")
	logger := scribe.NewLogger(os.Stdout)

	checksumCalculator := fs.NewChecksumCalculator()
	resolver := npm.NewBuildProcessResolver(executable, checksumCalculator, &logger)
	clock := npm.NewClock(time.Now)

	packit.Build(npm.Build(resolver, clock, &logger))
}
