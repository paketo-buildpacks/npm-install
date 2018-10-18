package integration

import (
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/npm-cnb/detect"

	"github.com/buildpack/libbuildpack"
	"github.com/cloudfoundry/dagger"
	. "github.com/onsi/gomega"
)

func TestIntegration(t *testing.T) {
	RegisterTestingT(t)
	spec.Run(t, "integration", testIntegration, spec.Report(report.Terminal{}))
}

func testIntegration(t *testing.T, when spec.G, it spec.S) {
	var (
		rootDir string
		dagg    *dagger.Dagger
	)

	it.Before(func() {
		var err error

		rootDir, err = dagger.FindRoot()
		Expect(err).ToNot(HaveOccurred())

		dagg, err = dagger.NewDagger(rootDir)
		Expect(err).ToNot(HaveOccurred())

	})

	it.After(func() {
		dagg.Destroy()
	})

	it("should run detect", func() {
		detectResult, err := dagg.Detect(
			filepath.Join(rootDir, "fixtures", "simple_app"),
			dagger.Order{
				Groups: []dagger.Group{
					{
						[]libbuildpack.BuildpackInfo{
							{
								ID:      "org.cloudfoundry.buildpacks.npm",
								Version: "0.0.1",
							},
						},
					},
				},
			})

		Expect(err).ToNot(HaveOccurred())

		Expect(len(detectResult.Group.Buildpacks)).To(Equal(1))
		Expect(detectResult.Group.Buildpacks[0].ID).To(Equal("org.cloudfoundry.buildpacks.npm"))
		Expect(detectResult.Group.Buildpacks[0].Version).To(Equal("0.0.1"))

		Expect(len(detectResult.BuildPlan)).To(Equal(2))

		Expect(detectResult.BuildPlan).To(HaveKey(detect.NodeDependency))
		Expect(detectResult.BuildPlan[detect.NodeDependency].Version).To(Equal("~10"))
		Expect(len(detectResult.BuildPlan[detect.NodeDependency].Metadata)).To(Equal(2))
		Expect(detectResult.BuildPlan[detect.NodeDependency].Metadata["build"]).To(BeTrue())
		Expect(detectResult.BuildPlan[detect.NodeDependency].Metadata["launch"]).To(BeTrue())

		Expect(detectResult.BuildPlan).To(HaveKey(detect.ModulesDependency))
		Expect(len(detectResult.BuildPlan[detect.ModulesDependency].Metadata)).To(Equal(1))
		Expect(detectResult.BuildPlan[detect.ModulesDependency].Metadata["launch"]).To(BeTrue())
	})

	it("should run build", func() {
		group := dagger.Group{
			Buildpacks: []libbuildpack.BuildpackInfo{
				{
					ID:      "org.cloudfoundry.buildpacks.npm",
					Version: "0.0.1",
				},
			},
		}
		buildResult, err := dagg.Build(filepath.Join(rootDir, "fixtures", "simple_app"), group, libbuildpack.BuildPlan{})
		Expect(err).ToNot(HaveOccurred())

		metadata, found, err := buildResult.GetLaunchMetadata()
		Expect(err).ToNot(HaveOccurred())
		Expect(found).To(Equal(true))

		Expect(len(metadata.Processes)).To(Equal(1))
		Expect(metadata.Processes[0].Type).To(Equal("web"))
		Expect(metadata.Processes[0].Command).To(Equal("npm start"))
	})
}
