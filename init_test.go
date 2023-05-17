package npminstall_test

import (
	"testing"

	"github.com/onsi/gomega/format"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitNPMInstall(t *testing.T) {
	format.MaxLength = 0

	suite := spec.New("npm-install", spec.Report(report.Terminal{}))
	suite("Build", testBuild)
	suite("BuildProcessResolver", testBuildProcessResolver)
	suite("CIBuildProcess", testCIBuildProcess)
	suite("Detect", testDetect)
	suite("Environment", testEnvironment)
	suite("InstallBuildProcess", testInstallBuildProcess)
	suite("LinkedModuleResolver", testLinkedModuleResolver)
	suite("Linker", testLinker)
	suite("PackageJSONParser", testPackageJSONParser)
	suite("PackageManangerConfigurationManager", testPackageManagerConfigurationManager)
	suite("PruneBuildProcess", testPruneBuildProcess)
	suite("RebuildBuildProcess", testRebuildBuildProcess)
	suite("UpdateNpmCacheLayer", testUpdateNpmCache)
	suite.Run(t)
}
