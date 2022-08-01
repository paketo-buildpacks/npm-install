package internal_test

import (
	"os"
	"path/filepath"
	"testing"

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
	)

	it.Before(func() {
		var err error
		layerDir, err = os.MkdirTemp("", "layerDir")
		Expect(err).NotTo(HaveOccurred())

		appDir, err = os.MkdirTemp("", "appDir")
		Expect(err).NotTo(HaveOccurred())

		tmpDir, err = os.MkdirTemp("", "tmp")
		Expect(err).NotTo(HaveOccurred())

		Expect(os.MkdirAll(filepath.Join(layerDir, "node_modules"), os.ModePerm)).To(Succeed())

		executablePath = filepath.Join(layerDir, "execd", "0-setup-symlinks")
		Expect(os.MkdirAll(filepath.Join(layerDir, "execd"), os.ModePerm)).To(Succeed())

		err = os.WriteFile(executablePath, []byte(""), 0600)
		Expect(err).NotTo(HaveOccurred())

	})

	it.After(func() {
		Expect(os.RemoveAll(layerDir)).To(Succeed())
		Expect(os.RemoveAll(appDir)).To(Succeed())
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	it("creates a symlink to the node_modules dir in the layer", func() {
		err := internal.Run(executablePath, appDir, tmpDir)
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
			err := internal.Run(executablePath, appDir, tmpDir)
			Expect(err).NotTo(HaveOccurred())

			link, err := os.Readlink(filepath.Join(tmpDir, "node_modules"))
			Expect(err).NotTo(HaveOccurred())
			Expect(link).To(Equal(filepath.Join(layerDir, "node_modules")))
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
				err := internal.Run(executablePath, appDir, tmpDir)
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})
	})
}
