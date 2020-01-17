package npm_test

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/npm-cnb/npm"
	"github.com/cloudfoundry/npm-cnb/npm/fakes"
	"github.com/cloudfoundry/packit/pexec"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testCIBuildProcess(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layerDir   string
		cacheDir   string
		workingDir string
		executable *fakes.Executable

		process npm.CIBuildProcess
	)

	context("Run", func() {
		it.Before(func() {
			var err error
			layerDir, err = ioutil.TempDir("", "layer")
			Expect(err).NotTo(HaveOccurred())

			cacheDir, err = ioutil.TempDir("", "layer")
			Expect(err).NotTo(HaveOccurred())

			workingDir, err = ioutil.TempDir("", "working-dir")
			Expect(err).NotTo(HaveOccurred())

			executable = &fakes.Executable{}

			process = npm.NewCIBuildProcess(executable)
		})

		it.After(func() {
			Expect(os.RemoveAll(layerDir)).To(Succeed())
			Expect(os.RemoveAll(cacheDir)).To(Succeed())
			Expect(os.RemoveAll(workingDir)).To(Succeed())
		})

		it("succeeds", func() {
			Expect(process.Run(layerDir, cacheDir, workingDir)).To(Succeed())

			Expect(executable.ExecuteCall.Receives.Execution).To(Equal(pexec.Execution{
				Args: []string{"ci", "--unsafe-perm", "--cache", cacheDir},
				Dir:  workingDir,
			}))

			path, err := os.Readlink(filepath.Join(workingDir, "node_modules"))
			Expect(err).NotTo(HaveOccurred())
			Expect(path).To(Equal(filepath.Join(layerDir, "node_modules")))
		})

		context("failure cases", func() {
			context("when the node_modules directory cannot be created", func() {
				it.Before(func() {
					Expect(os.Chmod(workingDir, 0000)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Chmod(workingDir, os.ModePerm)).To(Succeed())
				})

				it("returns an error", func() {
					err := process.Run(layerDir, cacheDir, workingDir)
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})

			context("when the node_modules directory cannot be moved to the layer", func() {
				it.Before(func() {
					Expect(os.Chmod(layerDir, 0000)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Chmod(layerDir, os.ModePerm)).To(Succeed())
				})

				it("returns an error", func() {
					err := process.Run(layerDir, cacheDir, workingDir)
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})

			context("when the executable fails", func() {
				it.Before(func() {
					executable.ExecuteCall.Returns.Err = errors.New("failed to execute")
				})

				it("returns an error", func() {
					err := process.Run(layerDir, cacheDir, workingDir)
					Expect(err).To(MatchError("failed to execute"))
				})
			})
		})
	})
}
