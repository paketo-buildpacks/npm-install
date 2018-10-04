package integration

import (
	"github.com/buildpack/libbuildpack"
	"github.com/cloudfoundry/npm-cnb/build"
	"path/filepath"

	"github.com/cloudfoundry/dagger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NPM buildpack", func() {
	var (
		rootDir string
		dagg    *dagger.Dagger
	)

	BeforeEach(func() {
		var err error

		rootDir, err = dagger.FindRoot()
		Expect(err).ToNot(HaveOccurred())

		dagg, err = dagger.NewDagger(rootDir)
		Expect(err).ToNot(HaveOccurred())

	})

	AfterEach(func() {
		dagg.Destroy()
	})

	It("should run detect", func() {
		detectResult, err := dagg.Detect(
			filepath.Join(rootDir, "fixtures", "simple_app"),
			dagger.Order{
				Groups: []dagger.Group{
					{
						[]libbuildpack.BuildpackInfo{
							{
								ID:      "org.cloudfoundry.buildpacks.npm",
								Version: "2.3.4",
							},
						},
					},
				},
			})

		Expect(err).ToNot(HaveOccurred())

		Expect(len(detectResult.Group.Buildpacks)).To(Equal(1))
		Expect(detectResult.Group.Buildpacks[0].ID).To(Equal("org.cloudfoundry.buildpacks.npm"))
		Expect(detectResult.Group.Buildpacks[0].Version).To(Equal("2.3.4"))

		Expect(len(detectResult.BuildPlan)).To(Equal(1))
		Expect(detectResult.BuildPlan).To(HaveKey("node"))
		Expect(detectResult.BuildPlan["node"].Provider).To(Equal("org.cloudfoundry.buildpacks.nodejs"))
		Expect(detectResult.BuildPlan["node"].Version).To(Equal("~10"))
	})

	It("should run build", func() {
		group := dagger.Group{
			Buildpacks: []libbuildpack.BuildpackInfo{
				{
					ID:      "org.cloudfoundry.buildpacks.npm",
					Version: "2.3.4",
				},
			},
		}
		plan := libbuildpack.BuildPlan{
			build.NodeDependency: libbuildpack.BuildPlanDependency{
				Provider: "org.cloudfoundry.buildpacks.nodejs",
				Version:  "~10",
			},
		}
		buildResult, err := dagg.Build(filepath.Join(rootDir, "fixtures", "simple_app"), group, plan)
		Expect(err).ToNot(HaveOccurred())

		metadata, found, err := buildResult.GetLaunchMetadata()
		Expect(err).ToNot(HaveOccurred())
		Expect(found).To(Equal(true))

		Expect(len(metadata.Processes)).To(Equal(1))
		Expect(metadata.Processes[0].Type).To(Equal("web"))
		Expect(metadata.Processes[0].Command).To(Equal("npm start"))
	})
})
