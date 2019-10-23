package integration_test

import (
	"os"
	"testing"

	"github.com/cloudfoundry/dagger"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

var (
	suite = spec.New("Integration", spec.Parallel(), spec.Report(report.Terminal{}))

	bpDir, npmURI, npmCachedURI, nodeURI, nodeCachedURI string
)

func TestIntegration(t *testing.T) {
	Expect := NewWithT(t).Expect

	var err error
	var nodeRepo string
	bpDir, err = dagger.FindBPRoot()
	Expect(err).NotTo(HaveOccurred())

	npmURI, err = dagger.PackageBuildpack(bpDir)
	Expect(err).ToNot(HaveOccurred())
	defer dagger.DeleteBuildpack(npmURI)

	npmCachedURI, _, err = dagger.PackageCachedBuildpack(bpDir)
	Expect(err).ToNot(HaveOccurred())
	defer dagger.DeleteBuildpack(npmCachedURI)

	nodeURI, err = dagger.GetLatestBuildpack("node-engine-cnb")
	Expect(err).ToNot(HaveOccurred())
	defer dagger.DeleteBuildpack(nodeURI)

	nodeRepo, err = dagger.GetLatestUnpackagedBuildpack("node-engine-cnb")
	Expect(err).ToNot(HaveOccurred())
	defer os.RemoveAll(nodeRepo)

	nodeCachedURI, _, err = dagger.PackageCachedBuildpack(nodeRepo)
	Expect(err).ToNot(HaveOccurred())
	defer dagger.DeleteBuildpack(nodeCachedURI)

	dagger.SyncParallelOutput(func() {
		suite.Run(t)
	})
}
