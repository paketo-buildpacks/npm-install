package modules_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitModules(t *testing.T) {
	suite := spec.New("modules", spec.Report(report.Terminal{}))

	suite("Contributor", testContributor)
	suite("Checksum", testChecksum)

	suite.Run(t)
}

type errReader struct{ err error }

func (r errReader) Read(p []byte) (int, error) { return 0, r.err }
