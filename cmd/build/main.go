package main

import (
	"os"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/npm-cnb/npm"
	"github.com/cloudfoundry/packit"
	"github.com/cloudfoundry/packit/fs"
	"github.com/cloudfoundry/packit/pexec"
	"github.com/cloudfoundry/packit/scribe"
)

func main() {
	executable := pexec.NewExecutable("npm", lager.NewLogger("npm"))
	logger := scribe.NewLogger(os.Stdout)

	packageJSONParser := npm.NewPackageJSONParser()
	checksumCalculator := fs.NewChecksumCalculator()
	resolver := npm.NewBuildProcessResolver(executable, packageJSONParser, checksumCalculator, logger)
	clock := npm.NewClock(time.Now)

	packit.Build(npm.Build(resolver, clock, logger))
}
