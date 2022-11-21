package npminstall_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitNPMInstall(t *testing.T) {
	suite := spec.New("npm-install", spec.Report(report.Terminal{}))
	suite("Build", testBuild)
	suite("BuildProcessResolver", testBuildProcessResolver)
	suite("CIBuildProcess", testCIBuildProcess)
	suite("Detect", testDetect)
	suite("InstallBuildProcess", testInstallBuildProcess)
	suite("PackageJSONParser", testPackageJSONParser)
	suite("PackageManangerConfigurationManager", testPackageManagerConfigurationManager)
	suite("ProjectPathParser", testProjectPathParser)
	suite("PruneBuildProcess", testPruneBuildProcess)
	suite("RebuildBuildProcess", testRebuildBuildProcess)
	suite("UpdateNpmCacheLayer", testUpdateNpmCache)
	suite.Run(t)
}
