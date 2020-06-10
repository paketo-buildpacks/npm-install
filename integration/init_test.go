package integration_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/paketo-buildpacks/occam"
	"github.com/paketo-buildpacks/packit/pexec"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

var (
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

	var config struct {
		NodeEngine string `json:"node-engine"`
	}

	file, err := os.Open("./../integration.json")
	Expect(err).NotTo(HaveOccurred())
	defer file.Close()

	Expect(json.NewDecoder(file).Decode(&config)).To(Succeed())

	root, err := filepath.Abs("./..")
	Expect(err).NotTo(HaveOccurred())

	buildpackStore := occam.NewBuildpackStore()

	version, err := GetGitVersion()
	Expect(err).NotTo(HaveOccurred())

	npmURI, err = buildpackStore.Get.WithVersion(version).Execute(root)
	Expect(err).ToNot(HaveOccurred())

	npmCachedURI, err = buildpackStore.Get.WithOfflineDependencies().WithVersion(version).Execute(root)
	Expect(err).ToNot(HaveOccurred())

	nodeURI, err = buildpackStore.Get.Execute(config.NodeEngine)
	Expect(err).ToNot(HaveOccurred())

	nodeCachedURI, err = buildpackStore.Get.WithOfflineDependencies().Execute(config.NodeEngine)
	Expect(err).ToNot(HaveOccurred())

	SetDefaultEventuallyTimeout(10 * time.Second)

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
	suite.Run(t)
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
