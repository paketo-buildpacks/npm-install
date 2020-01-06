package npm_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitNPM(t *testing.T) {
	suite := spec.New("npm", spec.Report(report.Terminal{}))
	suite("Build", testBuild)
	suite("Detect", testDetect)
	suite("NodePackageManager", testNodePackageManager)
	suite("PackageJSONParser", testPackageJSONParser)
	suite.Run(t)
}
