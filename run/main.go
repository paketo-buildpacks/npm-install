package main

import (
	"os"

	"github.com/paketo-buildpacks/npm"
	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/fs"
	"github.com/paketo-buildpacks/packit/pexec"
	"github.com/paketo-buildpacks/packit/scribe"
)

func main() {
	packageJSONParser := npm.NewPackageJSONParser()
	executable := pexec.NewExecutable("npm")
	logger := scribe.NewLogger(os.Stdout)
	checksumCalculator := fs.NewChecksumCalculator()
	resolver := npm.NewBuildProcessResolver(executable, checksumCalculator, &logger)

	packit.Run(
		npm.Detect(packageJSONParser),
		npm.Build(resolver, chronos.DefaultClock, &logger),
	)
}
