package npm_test

import (
	"errors"
	"io/ioutil"
	"os"
	"testing"

	"github.com/cloudfoundry/npm-cnb/npm"
	"github.com/cloudfoundry/npm-cnb/npm/fakes"
	"github.com/cloudfoundry/packit/pexec"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testInstallBuildProcess(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layerDir   string
		cacheDir   string
		workingDir string
		executable *fakes.Executable

		process npm.InstallBuildProcess
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

			process = npm.NewInstallBuildProcess(executable)
		})

		it.After(func() {
			Expect(os.RemoveAll(layerDir)).To(Succeed())
			Expect(os.RemoveAll(cacheDir)).To(Succeed())
			Expect(os.RemoveAll(workingDir)).To(Succeed())
		})

		it("succeeds", func() {
			Expect(process.Run(layerDir, cacheDir, workingDir)).To(Succeed())
			Expect(executable.ExecuteCall.Receives.Execution).To(Equal(pexec.Execution{
				Args: []string{"install", "--unsafe-perm", "--cache", cacheDir},
				Dir:  workingDir,
			}))
		})

		context("failure cases", func() {
			context("when unable to write node_modules directory in layer", func() {
				it.Before(func() {
					Expect(os.Chmod(layerDir, 0000)).To(Succeed())
				})

				it("fails", func() {
					err := process.Run(layerDir, cacheDir, workingDir)
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})

			context("when the node_modules directory cannot be symlinked into the working directory", func() {
				it.Before(func() {
					Expect(os.Chmod(workingDir, 0000)).To(Succeed())
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
