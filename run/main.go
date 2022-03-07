package main

import (
	"os"

	npminstall "github.com/paketo-buildpacks/npm-install"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/draft"
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
	logger := scribe.NewEmitter(os.Stdout)
	executable := pexec.NewExecutable("npm")
	buildProcessResolver := npminstall.NewBuildProcessResolver(executable, fs.NewChecksumCalculator(), npminstall.NewEnvironment(), scribe.NewLogger(os.Stdout))
	pruneBuildProcess := npminstall.NewPruneBuildProcess(executable, npminstall.NewEnvironment(), scribe.NewLogger(os.Stdout))
	entryResolver := draft.NewPlanner()
	sbomGenerator := SBOMGenerator{}
	packageManagerConfigurationManager := npminstall.NewPackageManagerConfigurationManager(servicebindings.NewResolver(), logger)

	packit.Run(
		npminstall.Detect(
			projectPathParser,
			packageJSONParser,
		),
		npminstall.Build(
			projectPathParser,
			entryResolver,
			packageManagerConfigurationManager,
			buildProcessResolver,
			pruneBuildProcess,
			chronos.DefaultClock,
			logger,
			sbomGenerator),
	)
}
