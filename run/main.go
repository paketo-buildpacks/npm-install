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
	projectPathParser := npminstall.NewProjectPathParser()
	packageJSONParser := npminstall.NewPackageJSONParser()
	executable := pexec.NewExecutable("npm")
	logger := scribe.NewLogger(os.Stdout)
	checksumCalculator := fs.NewChecksumCalculator()
	environment := npminstall.NewEnvironment(logger)
	resolver := npminstall.NewBuildProcessResolver(executable, checksumCalculator, environment, logger)

	packit.Run(
		npminstall.Detect(projectPathParser, packageJSONParser),
		npminstall.Build(
			projectPathParser,
			resolver,
			chronos.DefaultClock,
			environment,
			logger),
	)
}
