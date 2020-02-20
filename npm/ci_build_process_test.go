package npm_test

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/npm-cnb/npm"
	"github.com/cloudfoundry/npm-cnb/npm/fakes"
	"github.com/cloudfoundry/packit/pexec"
	"github.com/cloudfoundry/packit/scribe"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testCIBuildProcess(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		modulesDir    string
		cacheDir      string
		workingDir    string
		executable    *fakes.Executable
		summer        *fakes.Summer
		buffer        *bytes.Buffer
		commandOutput *bytes.Buffer

		process npm.CIBuildProcess
	)

	it.Before(func() {
		var err error
		modulesDir, err = ioutil.TempDir("", "modules")
		Expect(err).NotTo(HaveOccurred())

		cacheDir, err = ioutil.TempDir("", "cache")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		executable = &fakes.Executable{}
		summer = &fakes.Summer{}

		buffer = bytes.NewBuffer(nil)
		commandOutput = bytes.NewBuffer(nil)

		process = npm.NewCIBuildProcess(executable, summer, scribe.NewLogger(buffer))
	})

	it.After(func() {
		Expect(os.RemoveAll(modulesDir)).To(Succeed())
		Expect(os.RemoveAll(cacheDir)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("ShouldRun", func() {
		context("when the checksum matches the layer metadata shasum", func() {
			it.Before(func() {
				summer.SumCall.Returns.String = "some-cache-sha"
			})

			it("returns false", func() {
				run, sha, err := process.ShouldRun(workingDir, map[string]interface{}{
					"cache_sha": "some-cache-sha",
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(run).To(BeFalse())
				Expect(sha).To(BeEmpty())

				Expect(summer.SumCall.Receives.Path).To(Equal(filepath.Join(workingDir, "package-lock.json")))
			})
		})

		context("when the checksum does not match the layer metadata shasum", func() {
			it.Before(func() {
				summer.SumCall.Returns.String = "other-cache-sha"
			})

			it("returns false", func() {
				run, sha, err := process.ShouldRun(workingDir, map[string]interface{}{
					"cache_sha": "some-cache-sha",
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(run).To(BeTrue())
				Expect(sha).To(Equal("other-cache-sha"))

				Expect(summer.SumCall.Receives.Path).To(Equal(filepath.Join(workingDir, "package-lock.json")))
			})
		})

		context("when the layer metadata does not have a checksum", func() {
			it.Before(func() {
				summer.SumCall.Returns.String = "other-cache-sha"
			})

			it("returns false", func() {
				run, sha, err := process.ShouldRun(workingDir, nil)
				Expect(err).NotTo(HaveOccurred())
				Expect(run).To(BeTrue())
				Expect(sha).To(Equal("other-cache-sha"))

				Expect(summer.SumCall.Receives.Path).To(Equal(filepath.Join(workingDir, "package-lock.json")))
			})
		})

		context("failure cases", func() {
			context("when the there is an error in the checksummer process", func() {
				it.Before(func() {
					summer.SumCall.Returns.Error = errors.New("checksummer error")
				})

				it("returns an error", func() {
					_, _, err := process.ShouldRun(workingDir, nil)
					Expect(err).To(MatchError("checksummer error"))
				})
			})
		})
	})

	context("Run", func() {
		it("succeeds", func() {
			Expect(process.Run(modulesDir, cacheDir, workingDir)).To(Succeed())

			Expect(executable.ExecuteCall.Receives.Execution).To(Equal(pexec.Execution{
				Args:   []string{"ci", "--unsafe-perm", "--cache", cacheDir},
				Dir:    workingDir,
				Stdout: commandOutput,
				Stderr: commandOutput,
				Env:    append(os.Environ(), "NPM_CONFIG_PRODUCTION=true", "NPM_CONFIG_LOGLEVEL=error"),
			}))

			path, err := os.Readlink(filepath.Join(workingDir, "node_modules"))
			Expect(err).NotTo(HaveOccurred())
			Expect(path).To(Equal(filepath.Join(modulesDir, "node_modules")))
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
					err := process.Run(modulesDir, cacheDir, workingDir)
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})

			context("when the node_modules directory cannot be moved to the layer", func() {
				it.Before(func() {
					Expect(os.Chmod(modulesDir, 0000)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Chmod(modulesDir, os.ModePerm)).To(Succeed())
				})

				it("returns an error", func() {
					err := process.Run(modulesDir, cacheDir, workingDir)
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})

			context("when the executable fails", func() {
				it.Before(func() {
					executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
						fmt.Fprintln(execution.Stdout, "ci failure on stdout")
						fmt.Fprintln(execution.Stderr, "ci failure on stderr")
						return errors.New("failed to execute")
					}
				})

				it("returns an error", func() {
					err := process.Run(modulesDir, cacheDir, workingDir)
					Expect(buffer.String()).To(ContainSubstring("    ci failure on stdout\n    ci failure on stderr"))
					Expect(err).To(MatchError("npm ci failed: failed to execute"))
				})
			})
		})
	})
}
