package integration_test

import (
	"testing"

	"github.com/cloudfoundry/dagger"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

var (
	suite = spec.New("Integration", spec.Parallel(), spec.Report(report.Terminal{}))

	bpDir, npmURI, nodeURI string
)

func TestIntegration(t *testing.T) {
	Expect := NewWithT(t).Expect

	var err error
	bpDir, err = dagger.FindBPRoot()
	Expect(err).NotTo(HaveOccurred())

	npmURI, err = dagger.PackageBuildpack(bpDir)
	Expect(err).ToNot(HaveOccurred())
	defer dagger.DeleteBuildpack(npmURI)

	nodeURI, err = dagger.GetLatestBuildpack("node-engine-cnb")
	Expect(err).ToNot(HaveOccurred())
	defer dagger.DeleteBuildpack(nodeURI)

	dagger.SyncParallelOutput(func() {
		suite.Run(t)
	})
}
