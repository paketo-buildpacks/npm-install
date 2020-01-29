package main

import (
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/npm-cnb/npm"
	"github.com/cloudfoundry/packit"
	"github.com/cloudfoundry/packit/pexec"
)

func main() {
	executable := pexec.NewExecutable("npm", lager.NewLogger("npm"))
	packageJSONParser := npm.NewPackageJSONParser()
	checksumCalculator := npm.NewChecksumCalculator()
	resolver := npm.NewBuildProcessResolver(executable, packageJSONParser, checksumCalculator)
	clock := npm.NewClock(time.Now)

	packit.Build(npm.Build(resolver, clock))
}
