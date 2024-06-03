package npminstall_test

import (
	"os"
	"path/filepath"
	"testing"

	npminstall "github.com/paketo-buildpacks/npm-install"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testLinkedModuleResolver(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workspace, layerPath, otherLayerPath, tmpDir string
		resolver                                     npminstall.LinkedModuleResolver
	)

	it.Before(func() {
		var err error
		workspace, err = os.MkdirTemp("", "workspace")
		Expect(err).NotTo(HaveOccurred())

		layerPath, err = os.MkdirTemp("", "layer")
		Expect(err).NotTo(HaveOccurred())

		otherLayerPath, err = os.MkdirTemp("", "another-layer")
		Expect(err).NotTo(HaveOccurred())

		tmpDir, err = os.MkdirTemp("", "")
		Expect(err).NotTo(HaveOccurred())

		Expect(os.MkdirAll(filepath.Join(workspace, "src", "packages", "module-1"), os.ModePerm)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(workspace, "src", "packages", "module-1", "index.js"), nil, 0400)).To(Succeed())

		Expect(os.MkdirAll(filepath.Join(workspace, "workspaces", "example", "module-3"), os.ModePerm)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(workspace, "workspaces", "example", "module-3", "index.js"), nil, 0400)).To(Succeed())

		Expect(os.MkdirAll(filepath.Join(workspace, "module-5"), os.ModePerm)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(workspace, "module-5", "index.js"), nil, 0400)).To(Succeed())

		err = os.WriteFile(filepath.Join(workspace, "package-lock.json"), []byte(`{
			"packages": {
				"module-1": {
					"resolved": "src/packages/module-1",
					"link": true
				},
				"module-2": {
					"resolved": "http://example.com/module-2.tgz"
				},
				"module-3": {
					"resolved": "workspaces/example/module-3",
					"link": true
				},
				"module-4": {
					"resolved": "http://example.com/module-4.tgz"
				},
				"module-5": {
					"resolved": "module-5",
					"link": true
				}
			}
		}`), 0600)
		Expect(err).NotTo(HaveOccurred())

		resolver = npminstall.NewLinkedModuleResolver(npminstall.NewLinker(tmpDir))
	})

	it.After(func() {
		Expect(os.RemoveAll(workspace)).To(Succeed())
		Expect(os.RemoveAll(layerPath)).To(Succeed())
		Expect(os.RemoveAll(otherLayerPath)).To(Succeed())
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	context("Resolve", func() {
		it("resolves all linked modules in a package-lock.json", func() {
			err := resolver.Resolve(filepath.Join(workspace, "package-lock.json"), layerPath)
			Expect(err).NotTo(HaveOccurred())

			Expect(filepath.Join(layerPath, "module-5", "index.js")).To(BeARegularFile())
			Expect(filepath.Join(layerPath, "src", "packages", "module-1", "index.js")).To(BeARegularFile())
			Expect(filepath.Join(layerPath, "workspaces", "example", "module-3", "index.js")).To(BeARegularFile())

			link, err := os.Readlink(filepath.Join(workspace, "module-5"))
			Expect(err).NotTo(HaveOccurred())
			Expect(link).To(Equal(filepath.Join(tmpDir, "module-5")))

			link, err = os.Readlink(link)
			Expect(err).NotTo(HaveOccurred())
			Expect(link).To(Equal(filepath.Join(layerPath, "module-5")))

			link, err = os.Readlink(filepath.Join(workspace, "src", "packages", "module-1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(link).To(Equal(filepath.Join(tmpDir, "src", "packages", "module-1")))

			link, err = os.Readlink(link)
			Expect(err).NotTo(HaveOccurred())
			Expect(link).To(Equal(filepath.Join(layerPath, "src", "packages", "module-1")))

			link, err = os.Readlink(filepath.Join(workspace, "workspaces", "example", "module-3"))
			Expect(err).NotTo(HaveOccurred())
			Expect(link).To(Equal(filepath.Join(tmpDir, "workspaces", "example", "module-3")))

			link, err = os.Readlink(link)
			Expect(err).NotTo(HaveOccurred())
			Expect(link).To(Equal(filepath.Join(layerPath, "workspaces", "example", "module-3")))
		})

		context("failure cases", func() {
			context("when the lockfile cannot be opened", func() {
				it("returns an error", func() {
					err := resolver.Resolve("/no/such/path/to/package-lock.json", layerPath)
					Expect(err).To(MatchError(ContainSubstring(`failed to open "package-lock.json"`)))
				})
			})

			context("when the lockfile cannot be parsed", func() {
				it.Before(func() {
					Expect(os.WriteFile(filepath.Join(workspace, "package-lock.json"), []byte("%%%"), 0600)).To(Succeed())
				})

				it("returns an error", func() {
					err := resolver.Resolve(filepath.Join(workspace, "package-lock.json"), layerPath)
					Expect(err).To(MatchError(ContainSubstring(`failed to parse "package-lock.json"`)))
				})
			})

			context("when the destination cannot be scaffolded", func() {
				it.Before(func() {
					err := os.WriteFile(filepath.Join(workspace, "package-lock.json"), []byte(`{
						"packages": {
							"module": {
								"resolved": "src/packages/module",
								"link": true
							}
						}
					}`), 0600)
					Expect(err).NotTo(HaveOccurred())

					Expect(os.Mkdir(filepath.Join(layerPath, "sub-dir"), 0400)).To(Succeed())
				})

				it("returns an error", func() {
					err := resolver.Resolve(filepath.Join(workspace, "package-lock.json"), filepath.Join(layerPath, "sub-dir"))
					Expect(err).To(MatchError(ContainSubstring("failed to setup linked module directory scaffolding")))
				})
			})

			context("when the destination exists", func() {
				it.Before(func() {
					Expect(os.MkdirAll(filepath.Join(layerPath, "module-5"), os.ModePerm)).To(Succeed())
					Expect(os.WriteFile(filepath.Join(layerPath, "module-5", "index.js"), nil, 0400)).To(Succeed())
				})

				it("returns an error", func() {
					err := resolver.Resolve(filepath.Join(workspace, "package-lock.json"), layerPath)
					Expect(err).To(MatchError(ContainSubstring("failed to copy linked module directory to layer path")))
				})
			})

			context("when the destination cannot be linked", func() {
				it.Before(func() {
					Expect(os.Chmod(tmpDir, 0000)).To(Succeed())
				})

				it("returns an error", func() {
					err := resolver.Resolve(filepath.Join(workspace, "package-lock.json"), layerPath)
					Expect(err).To(MatchError(ContainSubstring("failed to symlink linked module directory")))
				})
			})
		})
	})

	context("Copy", func() {
		it.Before(func() {
			Expect(os.MkdirAll(filepath.Join(layerPath, "module-5"), os.ModePerm)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(layerPath, "module-5", "index.js"), nil, 0400)).To(Succeed())

			Expect(os.MkdirAll(filepath.Join(layerPath, "src", "packages", "module-1"), os.ModePerm)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(layerPath, "src", "packages", "module-1", "index.js"), nil, 0400)).To(Succeed())

			Expect(os.MkdirAll(filepath.Join(layerPath, "workspaces", "example", "module-3"), os.ModePerm)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(layerPath, "workspaces", "example", "module-3", "index.js"), nil, 0400)).To(Succeed())

		})

		it("resolves all linked modules in a package-lock.json", func() {
			err := resolver.Copy(filepath.Join(workspace, "package-lock.json"), layerPath, otherLayerPath)
			Expect(err).NotTo(HaveOccurred())

			Expect(filepath.Join(layerPath, "module-5", "index.js")).To(BeARegularFile())
			Expect(filepath.Join(layerPath, "src", "packages", "module-1", "index.js")).To(BeARegularFile())
			Expect(filepath.Join(layerPath, "workspaces", "example", "module-3", "index.js")).To(BeARegularFile())

		})

		context("failure cases", func() {
			context("when the lockfile cannot be opened", func() {
				it("returns an error", func() {
					err := resolver.Resolve("/no/such/path/to/package-lock.json", layerPath)
					Expect(err).To(MatchError(ContainSubstring(`failed to open "package-lock.json"`)))
				})
			})

			context("when the lockfile cannot be parsed", func() {
				it.Before(func() {
					Expect(os.WriteFile(filepath.Join(workspace, "package-lock.json"), []byte("%%%"), 0600)).To(Succeed())
				})

				it("returns an error", func() {
					err := resolver.Resolve(filepath.Join(workspace, "package-lock.json"), layerPath)
					Expect(err).To(MatchError(ContainSubstring(`failed to parse "package-lock.json"`)))
				})
			})

			context("when the destination cannot be scaffolded", func() {
				it.Before(func() {
					err := os.WriteFile(filepath.Join(workspace, "package-lock.json"), []byte(`{
						"packages": {
							"module": {
								"resolved": "src/packages/module",
								"link": true
							}
						}
					}`), 0600)
					Expect(err).NotTo(HaveOccurred())

					Expect(os.Mkdir(filepath.Join(otherLayerPath, "sub-dir"), 0400)).To(Succeed())
				})

				it("returns an error", func() {
					err := resolver.Resolve(filepath.Join(workspace, "package-lock.json"), filepath.Join(otherLayerPath, "sub-dir"))
					Expect(err).To(MatchError(ContainSubstring("failed to setup linked module directory scaffolding")))
				})
			})

			context("when the destination exists", func() {
				it.Before(func() {
					Expect(os.MkdirAll(filepath.Join(otherLayerPath, "module-5"), os.ModePerm)).To(Succeed())
					Expect(os.WriteFile(filepath.Join(otherLayerPath, "module-5", "index.js"), nil, 0400)).To(Succeed())
				})

				it("returns an error", func() {
					err := resolver.Resolve(filepath.Join(workspace, "package-lock.json"), layerPath)
					Expect(err).To(MatchError(ContainSubstring("failed to copy linked module directory to layer path")))
				})
			})

		})
	})
}
