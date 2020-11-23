package main

import (
	"os"

	npminstall "github.com/paketo-buildpacks/npm-install"
	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/fs"
	"github.com/paketo-buildpacks/packit/pexec"
	"github.com/paketo-buildpacks/packit/scribe"
)

func main() {
	packageJSONParser := npminstall.NewPackageJSONParser()
	executable := pexec.NewExecutable("npm")
	logger := scribe.NewLogger(os.Stdout)
	checksumCalculator := fs.NewChecksumCalculator()
	resolver := npminstall.NewBuildProcessResolver(executable, checksumCalculator, logger)

	packit.Run(
		npminstall.Detect(packageJSONParser),
		npminstall.Build(resolver, chronos.DefaultClock, logger),
	)
}
