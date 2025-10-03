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

var settings struct {
	Buildpacks struct {
		BuildPlan struct {
			Online string
		}
		NGINX struct {
			Online string
		}
		NodeEngine struct {
			Online  string
			Offline string
		}
		NodeRunScript struct {
			Online string
		}
		NPMInstall struct {
			Online string
		}
		NPMList struct {
			Online string
		}
		Cpython struct {
			Online string
		}
	}

	Extensions struct {
		UbiNodejsExtension struct {
			Online string
		}
	}

	Buildpack struct {
		ID   string
		Name string
	}

	Config struct {
		BuildPlan          string `json:"build-plan"`
		NodeEngine         string `json:"node-engine"`
		NodeRunScript      string `json:"node-run-script"`
		NGINX              string `json:"nginx"`
		UbiNodejsExtension string `json:"ubi-nodejs-extension"`
		Cpython            string `json:"cpython"`
	}
}

func TestIntegration(t *testing.T) {
	format.MaxLength = 0
	SetDefaultEventuallyTimeout(30 * time.Second)

	var Expect = NewWithT(t).Expect

	file, err := os.Open("../integration.json")
	Expect(err).NotTo(HaveOccurred())
	defer file.Close()

	Expect(json.NewDecoder(file).Decode(&settings.Config)).To(Succeed())

	file, err = os.Open("../buildpack.toml")
	Expect(err).NotTo(HaveOccurred())

	_, err = toml.NewDecoder(file).Decode(&settings)
	Expect(err).NotTo(HaveOccurred())

	root, err := filepath.Abs("./..")
	Expect(err).NotTo(HaveOccurred())

	buildpackStore := occam.NewBuildpackStore()

	pack := occam.NewPack()

	builder, err := pack.Builder.Inspect.Execute()
	Expect(err).NotTo(HaveOccurred())

	isUbiBuilder := builder.BuilderName == "paketobuildpacks/builder-ubi8-buildpackless-base" || builder.BuilderName == "paketobuildpacks/ubi-9-builder-buildpackless"

	if isUbiBuilder {
		settings.Extensions.UbiNodejsExtension.Online, err = buildpackStore.Get.
			Execute(settings.Config.UbiNodejsExtension)
		Expect(err).ToNot(HaveOccurred())
	}

	if !isUbiBuilder {
		settings.Buildpacks.Cpython.Online, err = buildpackStore.Get.
			Execute(settings.Config.Cpython)
		Expect(err).ToNot(HaveOccurred())
	}

	settings.Buildpacks.BuildPlan.Online, err = buildpackStore.Get.
		Execute(settings.Config.BuildPlan)
	Expect(err).NotTo(HaveOccurred())

	settings.Buildpacks.NGINX.Online, err = buildpackStore.Get.
		Execute(settings.Config.NGINX)
	Expect(err).NotTo(HaveOccurred())

	settings.Buildpacks.NodeEngine.Online, err = buildpackStore.Get.
		Execute(settings.Config.NodeEngine)
	Expect(err).ToNot(HaveOccurred())

	settings.Buildpacks.NodeEngine.Offline, err = buildpackStore.Get.
		WithOfflineDependencies().
		Execute(settings.Config.NodeEngine)
	Expect(err).ToNot(HaveOccurred())

	settings.Buildpacks.NodeRunScript.Online, err = buildpackStore.Get.
		Execute(settings.Config.NodeRunScript)
	Expect(err).ToNot(HaveOccurred())

	settings.Buildpacks.NPMInstall.Online, err = buildpackStore.Get.
		WithVersion("1.2.3").
		Execute(root)
	Expect(err).ToNot(HaveOccurred())

	settings.Buildpacks.NPMList.Online = filepath.Join(root, "integration", "testdata", "npm-list-buildpack")

	suite := spec.New("Integration", spec.Parallel(), spec.Report(report.Terminal{}))
	suite("Caching", testCaching)
	suite("DevDependenciesDuringBuild", testDevDependenciesDuringBuild)
	suite("EmptyNodeModules", testEmptyNodeModules)
	suite("Logging", testLogging)
	suite("NativeModules", testNativeModules)
	suite("NodeModulesCache", testReact)
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
	suite("Workspaces", testWorkspaces)
	suite.Run(t)
}
