package main

import (
	"os"

	npminstall "github.com/paketo-buildpacks/npm-install"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/fs"
	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/paketo-buildpacks/packit/v2/sbom"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	"github.com/paketo-buildpacks/packit/v2/servicebindings"
)

type SBOMGenerator struct{}

func (s SBOMGenerator) Generate(path string) (sbom.SBOM, error) {
	return sbom.Generate(path)
}

func main() {
	projectPathParser := npminstall.NewProjectPathParser()
	packageJSONParser := npminstall.NewPackageJSONParser()
	executable := pexec.NewExecutable("npm")
	logger := scribe.NewEmitter(os.Stdout)
	checksumCalculator := fs.NewChecksumCalculator()
	environment := npminstall.NewEnvironment(scribe.NewLogger(os.Stdout))
	resolver := npminstall.NewBuildProcessResolver(executable, checksumCalculator, environment, scribe.NewLogger(os.Stdout))
	sbomGenerator := SBOMGenerator{}
	bindingResolver := servicebindings.NewResolver()

	packit.Run(
		npminstall.Detect(projectPathParser, packageJSONParser),
		npminstall.Build(
			projectPathParser,
			bindingResolver,
			resolver,
			chronos.DefaultClock,
			environment,
			logger,
			sbomGenerator),
	)
}
