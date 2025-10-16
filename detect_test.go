package npminstall_test

import (
	"os"
	"path/filepath"
	"testing"

	npminstall "github.com/paketo-buildpacks/npm-install"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testDetect(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		detect     packit.DetectFunc
		filePath   string
		workingDir string
	)

	it.Before(func() {
		workingDir = t.TempDir()
		filePath = filepath.Join(workingDir, "package.json")
		Expect(os.WriteFile(filePath, []byte(`{
			"engines": {
					"node": "1.2.3"
			}
		}`), 0600)).To(Succeed())

		t.Setenv("BP_NODE_PROJECT_PATH", "")

		detect = npminstall.Detect()
	})

	it("returns a plan that provides node_modules", func() {
		result, err := detect(packit.DetectContext{
			WorkingDir: workingDir,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Plan).To(Equal(packit.BuildPlan{
			Provides: []packit.BuildPlanProvision{
				{Name: npminstall.NodeModules},
			},
			Requires: []packit.BuildPlanRequirement{
				{
					Name: npminstall.Node,
					Metadata: npminstall.BuildPlanMetadata{
						Version:       "1.2.3",
						VersionSource: "package.json",
						Build:         true,
					},
				},
				{
					Name: npminstall.Npm,
					Metadata: npminstall.BuildPlanMetadata{
						Build: true,
					},
				},
			},
		}))

	})

	context("when the package.json does not declare a node engine version", func() {
		it.Before(func() {
			Expect(os.WriteFile(filePath, []byte(`{
			}`), 0600)).To(Succeed())
		})

		it("returns a plan that does not declare a node version", func() {
			result, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Plan).To(Equal(packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{Name: npminstall.NodeModules},
				},
				Requires: []packit.BuildPlanRequirement{
					{
						Name: npminstall.Node,
						Metadata: npminstall.BuildPlanMetadata{
							Build: true,
						},
					},
					{
						Name: npminstall.Npm,
						Metadata: npminstall.BuildPlanMetadata{
							Build: true,
						},
					},
				},
			}))
		})
	})

	context("when the package.json file does not exist", func() {
		it.Before(func() {
			Expect(os.Remove(filePath)).To(Succeed())
		})

		it("fails detection", func() {
			_, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).To(MatchError(packit.Fail.WithMessage("no 'package.json' found in project path %s", workingDir)))
		})
	})

	context("when $BP_NPM_INCLUDE_BUILD_PYTHON env variable is present", func() {
		it.Before(func() {
			Expect(os.WriteFile(filePath, []byte(`{
			}`), 0600)).To(Succeed())
		})

		it.After(func() {
			os.Unsetenv("BP_NPM_INCLUDE_BUILD_PYTHON")
		})

		it("has been set to true, it should include cpython buildpack", func() {
			os.Setenv("BP_NPM_INCLUDE_BUILD_PYTHON", "true")

			result, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Plan).To(Equal(packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{Name: npminstall.NodeModules},
				},
				Requires: []packit.BuildPlanRequirement{
					{
						Name: npminstall.Node,
						Metadata: npminstall.BuildPlanMetadata{
							Build: true,
						},
					},
					{
						Name: npminstall.Cpython,
						Metadata: npminstall.BuildPlanMetadata{
							Build:  true,
							Launch: false,
						},
					},
					{
						Name: npminstall.Npm,
						Metadata: npminstall.BuildPlanMetadata{
							Build: true,
						},
					},
				},
			}))

		})

		it("has no value, it should include cpython buildpack", func() {
			os.Setenv("BP_NPM_INCLUDE_BUILD_PYTHON", "")

			result, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Plan).To(Equal(packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{Name: npminstall.NodeModules},
				},
				Requires: []packit.BuildPlanRequirement{
					{
						Name: npminstall.Node,
						Metadata: npminstall.BuildPlanMetadata{
							Build: true,
						},
					},
					{
						Name: npminstall.Cpython,
						Metadata: npminstall.BuildPlanMetadata{
							Build:  true,
							Launch: false,
						},
					},
					{
						Name: npminstall.Npm,
						Metadata: npminstall.BuildPlanMetadata{
							Build: true,
						},
					},
				},
			}))
		})

		it("has been set to false, it does not include cpython buildpack", func() {
			os.Setenv("BP_NPM_INCLUDE_BUILD_PYTHON", "false")

			result, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Plan).To(Equal(packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{Name: npminstall.NodeModules},
				},
				Requires: []packit.BuildPlanRequirement{
					{
						Name: npminstall.Node,
						Metadata: npminstall.BuildPlanMetadata{
							Build: true,
						},
					},
					{
						Name: npminstall.Npm,
						Metadata: npminstall.BuildPlanMetadata{
							Build: true,
						},
					},
				},
			}))
		})

		it("has been set to a random string, it does not include cpython buildpack", func() {
			os.Setenv("BP_NPM_INCLUDE_BUILD_PYTHON", "random-string")

			result, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Plan).To(Equal(packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{Name: npminstall.NodeModules},
				},
				Requires: []packit.BuildPlanRequirement{
					{
						Name: npminstall.Node,
						Metadata: npminstall.BuildPlanMetadata{
							Build: true,
						},
					},
					{
						Name: npminstall.Npm,
						Metadata: npminstall.BuildPlanMetadata{
							Build: true,
						},
					},
				},
			}))
		})
	})

	context("failure cases", func() {
		context("when the package.json parser fails", func() {
			it.Before(func() {
				Expect(os.WriteFile(filePath, []byte(`%%%`), 0600)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := detect(packit.DetectContext{
					WorkingDir: workingDir,
				})
				Expect(err).To(MatchError("unable to decode package.json invalid character '%' looking for beginning of value"))
			})
		})

		context("when the project path parser fails", func() {
			it.Before(func() {
				t.Setenv("BP_NODE_PROJECT_PATH", "does_not_exist")
			})

			it("returns an error", func() {
				_, err := detect(packit.DetectContext{
					WorkingDir: "/working-dir",
				})
				Expect(err).To(MatchError("could not find project path \"/working-dir/does_not_exist\": stat /working-dir/does_not_exist: no such file or directory"))
			})
		})
	})
}
