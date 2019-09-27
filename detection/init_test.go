package detection_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitDetection(t *testing.T) {
	suite := spec.New("detection", spec.Report(report.Terminal{}))

	suite("GetNodeVersion", testNodeVersion)
	suite("Plan", testPlan)

	suite.Run(t)
}
