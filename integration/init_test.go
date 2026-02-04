package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
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
		NodeEngine struct {
			Online string
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
		UbiNodejsExtension string `json:"ubi-nodejs-extension"`
		Cpython            string `json:"cpython"`
	}
}

func TestIntegration(t *testing.T) {
	format.MaxLength = 0
	SetDefaultEventuallyTimeout(30 * time.Second)

	var Expect = NewWithT(t).Expect

	integrationFile, err := os.Open("../integration.json")
	Expect(err).NotTo(HaveOccurred())
	Expect(json.NewDecoder(integrationFile).Decode(&settings.Config)).To(Succeed())
	Expect(integrationFile.Close()).To(Succeed())

	buildpackFile, err := os.Open("../buildpack.toml")
	Expect(err).NotTo(HaveOccurred())
	_, err = toml.NewDecoder(buildpackFile).Decode(&settings)
	Expect(err).NotTo(HaveOccurred())
	Expect(buildpackFile.Close()).To(Succeed())

	root, err := filepath.Abs("./..")
	Expect(err).NotTo(HaveOccurred())

	buildpackStore := occam.NewBuildpackStore()

	pack := occam.NewPack()

	builder, err := pack.Builder.Inspect.Execute()
	Expect(err).NotTo(HaveOccurred())

	isUbiBuilder := regexp.MustCompile(`ubi`).MatchString(builder.BuilderName)

	if isUbiBuilder {
		Expect(occam.NewDocker().Pull.Execute(settings.Config.UbiNodejsExtension)).To(Succeed())
		settings.Extensions.UbiNodejsExtension.Online = settings.Config.UbiNodejsExtension
	} else {
		Expect(occam.NewDocker().Pull.Execute(settings.Config.Cpython)).To(Succeed())
		settings.Buildpacks.Cpython.Online = settings.Config.Cpython
	}

	Expect(occam.NewDocker().Pull.Execute(settings.Config.BuildPlan)).To(Succeed())
	settings.Buildpacks.BuildPlan.Online = settings.Config.BuildPlan

	Expect(occam.NewDocker().Pull.Execute(settings.Config.NodeEngine)).To(Succeed())
	settings.Buildpacks.NodeEngine.Online = settings.Config.NodeEngine

	Expect(occam.NewDocker().Pull.Execute(settings.Config.NodeRunScript)).To(Succeed())
	settings.Buildpacks.NodeRunScript.Online = settings.Config.NodeRunScript

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
