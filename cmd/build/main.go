package main

import (
	"os"
	"time"

	"github.com/cloudfoundry/npm-cnb/npm"
	"github.com/cloudfoundry/packit"
	"github.com/cloudfoundry/packit/fs"
	"github.com/cloudfoundry/packit/pexec"
	"github.com/cloudfoundry/packit/scribe"
)

func main() {
	executable := pexec.NewExecutable("npm")
	logger := scribe.NewLogger(os.Stdout)

	checksumCalculator := fs.NewChecksumCalculator()
	resolver := npm.NewBuildProcessResolver(executable, checksumCalculator, logger)
	clock := npm.NewClock(time.Now)

	packit.Build(npm.Build(resolver, clock, logger))
}
