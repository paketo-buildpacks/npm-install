package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
)

var (
	buildpackURI        string
	buildpackOfflineURI string
	nodeURI             string
	nodeOfflineURI      string
	buildPlanURI        string
	npmList             string
	buildpackInfo       struct {
		Buildpack struct {
			ID   string
			Name string
		}
	}
)

func TestIntegration(t *testing.T) {
	format.MaxLength = 0
	SetDefaultEventuallyTimeout(30 * time.Second)

	var (
		Expect = NewWithT(t).Expect
		err    error
	)

	var config struct {
		NodeEngine string `json:"node-engine"`
		BuildPlan  string `json:"build-plan"`
	}

	file, err := os.Open("../integration.json")
	Expect(err).NotTo(HaveOccurred())
	defer file.Close()

	Expect(json.NewDecoder(file).Decode(&config)).To(Succeed())

	file, err = os.Open("../buildpack.toml")
	Expect(err).NotTo(HaveOccurred())

	_, err = toml.NewDecoder(file).Decode(&buildpackInfo)
	Expect(err).NotTo(HaveOccurred())

	root, err := filepath.Abs("./..")
	Expect(err).NotTo(HaveOccurred())

	buildpackStore := occam.NewBuildpackStore()

	buildpackURI, err = buildpackStore.Get.
		WithVersion("1.2.3").
		Execute(root)
	Expect(err).ToNot(HaveOccurred())

	buildpackOfflineURI, err = buildpackStore.Get.
		WithOfflineDependencies().
		WithVersion("1.2.3").
		Execute(root)
	Expect(err).ToNot(HaveOccurred())

	nodeURI, err = buildpackStore.Get.
		Execute(config.NodeEngine)
	Expect(err).ToNot(HaveOccurred())

	nodeOfflineURI, err = buildpackStore.Get.
		WithOfflineDependencies().
		Execute(config.NodeEngine)
	Expect(err).ToNot(HaveOccurred())

	buildPlanURI, err = buildpackStore.Get.
		Execute(config.BuildPlan)
	Expect(err).NotTo(HaveOccurred())

	npmList = filepath.Join(root, "integration", "testdata", "npm-list-buildpack")

	suite := spec.New("Integration", spec.Parallel(), spec.Report(report.Terminal{}))
	suite("Caching", testCaching)
	suite("DevDependenciesDuringBuild", testDevDependenciesDuringBuild)
	suite("EmptyNodeModules", testEmptyNodeModules)
	suite("Logging", testLogging)
	suite("NoNodeModules", testNoNodeModules)
	suite("Npmrc", testNpmrc)
	suite("PackageLockMismatch", testPackageLockMismatch)
	suite("PrePostScriptsRebuild", testPrePostScriptRebuild)
	suite("ProjectPath", testProjectPath)
	suite("Restart", testRestart)
	suite("SimpleApp", testSimpleApp)
	suite("UnmetDependencies", testUnmetDependencies)
	suite("Vendored", testVendored)
	suite("VendoredWithBinaries", testVendoredWithBinaries)
	suite("Versioning", testVersioning)
	suite.Run(t)
}
