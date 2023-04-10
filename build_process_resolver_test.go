package npminstall_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	npminstall "github.com/paketo-buildpacks/npm-install"
	"github.com/paketo-buildpacks/npm-install/fakes"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testBuildProcessResolver(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir string

		rebuild *fakes.BuildProcess
		install *fakes.BuildProcess
		ci      *fakes.BuildProcess

		resolver npminstall.BuildProcessResolver

		buffer *bytes.Buffer
	)

	it.Before(func() {
		var err error
		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		buffer = bytes.NewBuffer(nil)
		logger := scribe.NewLogger(buffer)

		rebuild = &fakes.BuildProcess{}
		rebuild.ShouldRunCall.Returns.Sha = "rebuild-sha"

		install = &fakes.BuildProcess{}
		install.ShouldRunCall.Returns.Sha = "install-sha"

		ci = &fakes.BuildProcess{}
		ci.ShouldRunCall.Returns.Sha = "ci-sha"

		resolver = npminstall.NewBuildProcessResolver(logger, rebuild, install, ci)
	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("the build process is install", func() {
		context("there is no node_modules, package-lock.json, or cache", func() {
			it("returns the install process", func() {
				buildProcess, cacheUsed, err := resolver.Resolve(workingDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(cacheUsed).To(BeFalse())

				Expect(buildProcess).To(Equal(install))

				Expect(buffer.String()).To(ContainSubstring("Selected NPM build process: 'npm install'"))
			})
		})

		context("there is no node_modules, package-lock.json but there is a cache", func() {
			it.Before(func() {
				Expect(os.MkdirAll(filepath.Join(workingDir, "npm-cache"), os.ModePerm)).To(Succeed())
			})

			it("returns the install process", func() {
				buildProcess, cacheUsed, err := resolver.Resolve(workingDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(cacheUsed).To(BeTrue())

				Expect(buildProcess).To(Equal(install))
			})
		})
	})

	context("the build process is rebuild", func() {
		context("there is no package-lock.json or cache but there is node_modules", func() {
			it.Before(func() {
				Expect(os.MkdirAll(filepath.Join(workingDir, "node_modules"), os.ModePerm)).To(Succeed())
			})

			it("returns the rebuild process", func() {
				buildProcess, cacheUsed, err := resolver.Resolve(workingDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(cacheUsed).To(BeFalse())

				Expect(buildProcess).To(Equal(rebuild))

				Expect(buffer.String()).To(ContainSubstring("Selected NPM build process: 'npm rebuild'"))
			})
		})

		context("there is no package-lock.json but both node_modules and a cache are present", func() {
			it.Before(func() {
				Expect(os.MkdirAll(filepath.Join(workingDir, "node_modules"), os.ModePerm)).To(Succeed())

				Expect(os.MkdirAll(filepath.Join(workingDir, "npm-cache"), os.ModePerm)).To(Succeed())
			})

			it("returns the rebuild process", func() {
				buildProcess, cacheUsed, err := resolver.Resolve(workingDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(cacheUsed).To(BeTrue())

				Expect(buildProcess).To(Equal(rebuild))
			})
		})

		context("there is no cache but node_modules and a package-lock.json are present", func() {
			it.Before(func() {
				Expect(os.MkdirAll(filepath.Join(workingDir, "node_modules"), os.ModePerm)).To(Succeed())

				err := os.WriteFile(filepath.Join(workingDir, "package-lock.json"), []byte("some-content"), 0644)
				Expect(err).NotTo(HaveOccurred())
			})

			it("returns the rebuild process", func() {
				buildProcess, cacheUsed, err := resolver.Resolve(workingDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(cacheUsed).To(BeFalse())

				Expect(buildProcess).To(Equal(rebuild))
			})
		})
	})

	context("the build process is ci", func() {
		context("there is no node_modules or cache but there is a package-lock.json", func() {
			it.Before(func() {
				err := os.WriteFile(filepath.Join(workingDir, "package-lock.json"), []byte("some-content"), 0644)
				Expect(err).NotTo(HaveOccurred())
			})

			it("returns the ci process", func() {
				buildProcess, cacheUsed, err := resolver.Resolve(workingDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(cacheUsed).To(BeFalse())

				Expect(buildProcess).To(Equal(ci))

				Expect(buffer.String()).To(ContainSubstring("Selected NPM build process: 'npm ci'"))
			})
		})

		context("there is no node_modules but the package-lock.json and cache are present", func() {
			it.Before(func() {
				err := os.WriteFile(filepath.Join(workingDir, "package-lock.json"), []byte("some-content"), 0644)
				Expect(err).NotTo(HaveOccurred())

				Expect(os.MkdirAll(filepath.Join(workingDir, "npm-cache"), os.ModePerm)).To(Succeed())
			})

			it("returns the ci process", func() {
				buildProcess, cacheUsed, err := resolver.Resolve(workingDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(cacheUsed).To(BeTrue())

				Expect(buildProcess).To(Equal(ci))
			})
		})

		context("the package-lock.json, node_modules, and cache are present", func() {
			it.Before(func() {
				Expect(os.MkdirAll(filepath.Join(workingDir, "node_modules"), os.ModePerm)).To(Succeed())

				err := os.WriteFile(filepath.Join(workingDir, "package-lock.json"), []byte("some-content"), 0644)
				Expect(err).NotTo(HaveOccurred())

				Expect(os.MkdirAll(filepath.Join(workingDir, "npm-cache"), os.ModePerm)).To(Succeed())
			})

			it("returns the ci process", func() {
				buildProcess, cacheUsed, err := resolver.Resolve(workingDir)
				Expect(err).NotTo(HaveOccurred())

				Expect(cacheUsed).To(BeTrue())
				Expect(buildProcess).To(Equal(ci))
			})
		})
	})

	context("output cases", func() {
		context("when there is a package-lock.json", func() {
			it.Before(func() {
				err := os.WriteFile(filepath.Join(workingDir, "package-lock.json"), []byte("some-content"), 0644)
				Expect(err).NotTo(HaveOccurred())
			})

			it("outputs that package-lock.json was found", func() {
				_, _, err := resolver.Resolve(workingDir)
				Expect(err).NotTo(HaveOccurred())

				Expect(buffer.String()).To(MatchRegexp(`package-lock.json\s+-> "Found"`))
			})
		})

		context("when there is a node_modules directory", func() {
			it.Before(func() {
				Expect(os.MkdirAll(filepath.Join(workingDir, "node_modules"), os.ModePerm)).To(Succeed())
			})

			it("outputs that package-lock.json was found", func() {
				_, _, err := resolver.Resolve(workingDir)
				Expect(err).NotTo(HaveOccurred())

				Expect(buffer.String()).To(MatchRegexp(`node_modules\s+-> "Found"`))
			})
		})

		context("when there is a node_modules directory", func() {
			it.Before(func() {
				Expect(os.MkdirAll(filepath.Join(workingDir, "npm-cache"), os.ModePerm)).To(Succeed())
			})

			it("outputs that package-lock.json was found", func() {
				_, _, err := resolver.Resolve(workingDir)
				Expect(err).NotTo(HaveOccurred())

				Expect(buffer.String()).To(MatchRegexp(`npm-cache\s+-> "Found"`))
			})
		})
	})

	context("failure cases", func() {
		var resolver npminstall.BuildProcessResolver

		it.Before(func() {
			var err error
			workingDir, err = os.MkdirTemp("", "working-dir")
			Expect(err).NotTo(HaveOccurred())

			logger := scribe.NewLogger(bytes.NewBuffer(nil))
			resolver = npminstall.NewBuildProcessResolver(logger, rebuild, install, ci)
		})

		it.After(func() {
			Expect(os.RemoveAll(workingDir)).To(Succeed())
		})

		context("Resolve", func() {
			context("when the working directory is unreadable", func() {
				it.Before(func() {
					Expect(os.Chmod(workingDir, 0000)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Chmod(workingDir, os.ModePerm)).To(Succeed())
				})

				it("returns an error", func() {
					_, _, err := resolver.Resolve(workingDir)
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})
		})
	})
}
