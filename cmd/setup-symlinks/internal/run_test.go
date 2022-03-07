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
	)

	it.Before(func() {
		var err error
		layerDir, err = os.MkdirTemp("", "layerDir")
		Expect(err).NotTo(HaveOccurred())

		appDir, err = os.MkdirTemp("", "appDir")
		Expect(err).NotTo(HaveOccurred())

		Expect(os.MkdirAll(filepath.Join(layerDir, "node_modules"), os.ModePerm)).To(Succeed())

		executablePath = filepath.Join(layerDir, "execd", "0-setup-symlinks")
		Expect(os.MkdirAll(filepath.Join(layerDir, "execd"), os.ModePerm)).To(Succeed())

		err = os.WriteFile(executablePath, []byte(""), 0600)
		Expect(err).NotTo(HaveOccurred())

		Expect(os.Symlink("build-modules", filepath.Join(appDir, "node_modules"))).To(Succeed())

	})

	it.After(func() {
		Expect(os.RemoveAll(layerDir)).To(Succeed())
		Expect(os.RemoveAll(appDir)).To(Succeed())
	})

	it("creates a symlink to the node_modules dir in the layer", func() {
		err := internal.Run(executablePath, appDir)
		Expect(err).NotTo(HaveOccurred())

		link, err := os.Readlink(filepath.Join(appDir, "node_modules"))
		Expect(err).NotTo(HaveOccurred())
		Expect(link).To(Equal(filepath.Join(layerDir, "node_modules")))
	})

	context("failure cases", func() {
		context("when the app dir node_modules cannot be removed", func() {
			it.Before(func() {
				Expect(os.Chmod(appDir, 0444)).To(Succeed())
			})
			it.After(func() {
				Expect(os.Chmod(appDir, os.ModePerm)).To(Succeed())
			})
			it("returns an error", func() {
				err := internal.Run(executablePath, appDir)
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})
	})
}
