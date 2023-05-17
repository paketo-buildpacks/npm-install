package internal_test

import (
	"os"
	"path/filepath"
	"testing"

	npminstall "github.com/paketo-buildpacks/npm-install"
	"github.com/paketo-buildpacks/npm-install/cmd/setup-symlinks/internal"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

func TestUnitSetupSymlinks(t *testing.T) {
	suite := spec.New("cmd/setup-symlinks/internal", spec.Report(report.Terminal{}))
	suite("Run", testRun)
	suite.Run(t)
}

func testRun(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layerDir       string
		executablePath string
		appDir         string
		tmpDir         string
		resolver       npminstall.LinkedModuleResolver
	)

	it.Before(func() {
		var err error
		layerDir, err = os.MkdirTemp("", "layerDir")
		Expect(err).NotTo(HaveOccurred())

		appDir, err = os.MkdirTemp("", "appDir")
		Expect(err).NotTo(HaveOccurred())

		tmpDir, err = os.MkdirTemp("", "tmp")
		Expect(err).NotTo(HaveOccurred())

		Expect(os.Symlink(filepath.Join(tmpDir, "node_modules"), filepath.Join(appDir, "node_modules"))).To(Succeed())

		Expect(os.MkdirAll(filepath.Join(layerDir, "node_modules"), os.ModePerm)).To(Succeed())

		executablePath = filepath.Join(layerDir, "execd", "0-setup-symlinks")
		Expect(os.MkdirAll(filepath.Join(layerDir, "execd"), os.ModePerm)).To(Succeed())

		err = os.WriteFile(executablePath, []byte(""), 0600)
		Expect(err).NotTo(HaveOccurred())

		Expect(os.MkdirAll(filepath.Join(appDir, "src", "packages", "module-1"), os.ModePerm)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(appDir, "src", "packages", "module-1", "index.js"), nil, 0400)).To(Succeed())

		Expect(os.MkdirAll(filepath.Join(appDir, "workspaces", "example", "module-3"), os.ModePerm)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(appDir, "workspaces", "example", "module-3", "index.js"), nil, 0400)).To(Succeed())

		Expect(os.MkdirAll(filepath.Join(appDir, "module-5"), os.ModePerm)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(appDir, "module-5", "index.js"), nil, 0400)).To(Succeed())

		resolver = npminstall.NewLinkedModuleResolver(npminstall.NewLinker(tmpDir))
		err = os.WriteFile(filepath.Join(appDir, "package-lock.json"), []byte(`{
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

	})

	it.After(func() {
		Expect(os.RemoveAll(layerDir)).To(Succeed())
		Expect(os.RemoveAll(appDir)).To(Succeed())
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	it("creates a symlink to the node_modules dir in the layer", func() {
		err := resolver.Resolve(filepath.Join(appDir, "package-lock.json"), layerDir)
		Expect(err).NotTo(HaveOccurred())

		err = internal.Run(executablePath, appDir, resolver)
		Expect(err).NotTo(HaveOccurred())

		link, err := os.Readlink(filepath.Join(tmpDir, "node_modules"))
		Expect(err).NotTo(HaveOccurred())
		Expect(link).To(Equal(filepath.Join(layerDir, "node_modules")))
	})

	context("when the symlink already exists", func() {
		it.Before(func() {
			Expect(os.Symlink("some-location", filepath.Join(tmpDir, "node_modules"))).To(Succeed())
		})

		it("replaces it", func() {
			err := resolver.Resolve(filepath.Join(appDir, "package-lock.json"), layerDir)
			Expect(err).NotTo(HaveOccurred())

			err = internal.Run(executablePath, appDir, resolver)
			Expect(err).NotTo(HaveOccurred())

			link, err := os.Readlink(filepath.Join(tmpDir, "node_modules"))
			Expect(err).NotTo(HaveOccurred())
			Expect(link).To(Equal(filepath.Join(layerDir, "node_modules")))
		})
	})

	context("when the symlink points to non-existent directory", func() {
		it.Before(func() {
			Expect(os.RemoveAll(filepath.Join(appDir, "node_modules"))).To(Succeed())
			Expect(os.Symlink(filepath.Join(tmpDir, "non-existing-tempdir", "node_modules"), filepath.Join(appDir, "node_modules"))).To(Succeed())
		})

		it("ensures the parent directory exists", func() {
			err := resolver.Resolve(filepath.Join(appDir, "package-lock.json"), layerDir)
			Expect(err).NotTo(HaveOccurred())

			err = internal.Run(executablePath, appDir, resolver)
			Expect(err).NotTo(HaveOccurred())

			link, err := os.Readlink(filepath.Join(tmpDir, "non-existing-tempdir", "node_modules"))
			Expect(err).NotTo(HaveOccurred())
			Expect(link).To(Equal(filepath.Join(layerDir, "node_modules")))
		})
	})

	context("when appDir contains workspace packages ", func() {

		it("ensures symlinks to the layer exist at launch time", func() {

			err := resolver.Resolve(filepath.Join(appDir, "package-lock.json"), layerDir)
			Expect(err).NotTo(HaveOccurred())

			err = internal.Run(executablePath, appDir, resolver)
			Expect(err).NotTo(HaveOccurred())

			link, err := os.Readlink(filepath.Join(appDir, "src", "packages", "module-1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(link).To(Equal(filepath.Join(tmpDir, "src", "packages", "module-1")))

			link, err = os.Readlink(link)
			Expect(err).NotTo(HaveOccurred())
			Expect(link).To(Equal(filepath.Join(layerDir, "src", "packages", "module-1")))

			link, err = os.Readlink(filepath.Join(appDir, "workspaces", "example", "module-3"))
			Expect(err).NotTo(HaveOccurred())
			Expect(link).To(Equal(filepath.Join(tmpDir, "workspaces", "example", "module-3")))

			link, err = os.Readlink(link)
			Expect(err).NotTo(HaveOccurred())
			Expect(link).To(Equal(filepath.Join(layerDir, "workspaces", "example", "module-3")))

			link, err = os.Readlink(filepath.Join(appDir, "module-5"))
			Expect(err).NotTo(HaveOccurred())
			Expect(link).To(Equal(filepath.Join(tmpDir, "module-5")))

			link, err = os.Readlink(link)
			Expect(err).NotTo(HaveOccurred())
			Expect(link).To(Equal(filepath.Join(layerDir, "module-5")))

		})
	})

	context("failure cases", func() {
		context("when the tmp dir node_modules cannot be removed", func() {
			it.Before(func() {
				Expect(os.Chmod(tmpDir, 0444)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(tmpDir, os.ModePerm)).To(Succeed())
			})

			it("returns an error", func() {
				err := internal.Run(executablePath, appDir, resolver)
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})
	})
}
