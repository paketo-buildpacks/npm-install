package integration_test

import (
	"os"
	"testing"

	"github.com/cloudfoundry/dagger"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

var (
	bpDir         string
	npmURI        string
	npmCachedURI  string
	nodeURI       string
	nodeCachedURI string
)

func TestIntegration(t *testing.T) {
	var (
		Expect = NewWithT(t).Expect
		err    error
	)

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

	nodeRepo, err := dagger.GetLatestUnpackagedBuildpack("node-engine-cnb")
	Expect(err).ToNot(HaveOccurred())
	defer os.RemoveAll(nodeRepo)

	nodeCachedURI, _, err = dagger.PackageCachedBuildpack(nodeRepo)
	Expect(err).ToNot(HaveOccurred())
	defer dagger.DeleteBuildpack(nodeCachedURI)

	suite := spec.New("Integration", spec.Parallel(), spec.Report(report.Terminal{}))
	suite("EmptyNodeModules", testEmptyNodeModules)
	suite("IncompleteNodeModules", testIncompleteNodeModules)
	suite("IncompletePackageJSON", testIncompletePackageJSON)
	suite("NoNodeModules", testNoNodeModules)
	suite("SimpleApp", testSimpleApp)
	suite("UnmetDependencies", testUnmetDependencies)
	suite("Vendored", testVendored)
	suite("Versioning", testVersioning)

	dagger.SyncParallelOutput(func() { suite.Run(t) })
}
