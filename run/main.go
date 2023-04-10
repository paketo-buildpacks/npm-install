package main

import (
	"log"
	"os"
	"path/filepath"

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
	environment, err := npminstall.ParseEnvironment(filepath.Join(os.Getenv("CNB_BUILDPACK_DIR"), "buildpack.toml"), os.Environ())
	if err != nil {
		log.Fatal(err)
	}

	logLevel, _ := environment.Lookup("BP_LOG_LEVEL")
	globalConfigPath, _ := environment.Lookup("NPM_CONFIG_GLOBALCONFIG")

	emitter := scribe.NewEmitter(os.Stdout).WithLevel(logLevel)
	logger := scribe.NewLogger(os.Stdout).WithLevel(logLevel)

	projectPathParser := npminstall.NewProjectPathParser(environment)
	npm := pexec.NewExecutable("npm")
	checksumCalculator := fs.NewChecksumCalculator()
	linker := npminstall.NewLinker(os.TempDir())

	packit.Run(
		npminstall.Detect(
			projectPathParser,
			npminstall.NewPackageJSONParser(),
		),
		npminstall.Build(
			projectPathParser,
			draft.NewPlanner(),
			npminstall.NewPackageManagerConfigurationManager(
				servicebindings.NewResolver(),
				emitter,
				globalConfigPath,
			),
			npminstall.NewBuildProcessResolver(
				logger,
				npminstall.NewRebuildBuildProcess(npm, checksumCalculator, environment, logger),
				npminstall.NewInstallBuildProcess(npm, environment, logger),
				npminstall.NewCIBuildProcess(npm, checksumCalculator, environment, logger),
			),
			npminstall.NewPruneBuildProcess(
				npm,
				environment,
				logger,
			),
			chronos.DefaultClock,
			emitter,
			SBOMGenerator{},
			linker,
			environment,
			npminstall.NewLinkedModuleResolver(linker),
		),
	)
}
