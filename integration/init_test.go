package integration_test

import (
	"fmt"
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

	npmCachedURI, _, err = dagger.PackageCachedBuildpack(bpDir)
	Expect(err).ToNot(HaveOccurred())

	nodeURI, err = dagger.GetLatestBuildpack("node-engine-cnb")
	Expect(err).ToNot(HaveOccurred())

	nodeRepo, err := dagger.GetLatestUnpackagedBuildpack("node-engine-cnb")
	Expect(err).ToNot(HaveOccurred())

	nodeCachedURI, _, err = dagger.PackageCachedBuildpack(nodeRepo)
	Expect(err).ToNot(HaveOccurred())

	// HACK: we need to fix dagger and the package.sh scripts so that this isn't required
	npmURI = fmt.Sprintf("%s.tgz", npmURI)
	npmCachedURI = fmt.Sprintf("%s.tgz", npmCachedURI)
	nodeCachedURI = fmt.Sprintf("%s.tgz", nodeCachedURI)

	defer dagger.DeleteBuildpack(npmURI)
	defer dagger.DeleteBuildpack(npmCachedURI)
	defer dagger.DeleteBuildpack(nodeURI)
	defer os.RemoveAll(nodeRepo)
	defer dagger.DeleteBuildpack(nodeCachedURI)

	suite := spec.New("Integration", spec.Parallel(), spec.Report(report.Terminal{}))
	suite("Caching", testCaching)
	suite("EmptyNodeModules", testEmptyNodeModules)
	suite("NoNodeModules", testNoNodeModules)
	suite("PrePostScriptsRebuild", testPrePostScriptRebuild)
	suite("SimpleApp", testSimpleApp)
	suite("UnmetDependencies", testUnmetDependencies)
	suite("Vendored", testVendored)
	suite("Versioning", testVersioning)

	dagger.SyncParallelOutput(func() { suite.Run(t) })
}
