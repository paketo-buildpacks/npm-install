package npm_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitNPM(t *testing.T) {
	suite := spec.New("npm", spec.Report(report.Terminal{}))
	suite("Build", testBuild)
	suite("BuildProcessResolver", testBuildProcessResolver)
	suite("CIBuildProcess", testCIBuildProcess)
	suite("Detect", testDetect)
	suite("InstallBuildProcess", testInstallBuildProcess)
	suite("PackageJSONParser", testPackageJSONParser)
	suite("RebuildBuildProcess", testRebuildBuildProcess)
	suite.Run(t)
}
