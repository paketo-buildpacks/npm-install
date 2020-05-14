package integration_test

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cloudfoundry/dagger"
	"github.com/paketo-buildpacks/occam"
	"github.com/paketo-buildpacks/packit/pexec"
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

	SetDefaultEventuallyTimeout(5 * time.Second)

	suite := spec.New("Integration", spec.Random(), spec.Report(report.Terminal{}))
	suite("Caching", testCaching)
	suite("EmptyNodeModules", testEmptyNodeModules, spec.Parallel())
	suite("Logging", testLogging, spec.Parallel())
	suite("NoNodeModules", testNoNodeModules, spec.Parallel())
	suite("PrePostScriptsRebuild", testPrePostScriptRebuild, spec.Parallel())
	suite("SimpleApp", testSimpleApp, spec.Parallel())
	suite("UnmetDependencies", testUnmetDependencies, spec.Parallel())
	suite("Vendored", testVendored, spec.Parallel())
	suite("VendoredWithBinaries", testVendoredWithBinaries, spec.Parallel())
	suite("Versioning", testVersioning, spec.Parallel())
	suite("Npmrc", testNpmrc, spec.Parallel())

	dagger.SyncParallelOutput(func() { suite.Run(t) })
}

func ContainerLogs(id string) func() string {
	docker := occam.NewDocker()

	return func() string {
		logs, _ := docker.Container.Logs.Execute(id)
		return logs.String()
	}
}

func GetGitVersion() (string, error) {
	gitExec := pexec.NewExecutable("git")
	revListOut := bytes.NewBuffer(nil)

	err := gitExec.Execute(pexec.Execution{
		Args:   []string{"rev-list", "--tags", "--max-count=1"},
		Stdout: revListOut,
	})
	if err != nil {
		return "", err
	}

	stdout := bytes.NewBuffer(nil)
	err = gitExec.Execute(pexec.Execution{
		Args:   []string{"describe", "--tags", strings.TrimSpace(revListOut.String())},
		Stdout: stdout,
	})
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(strings.TrimPrefix(stdout.String(), "v")), nil
}
