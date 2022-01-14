package npminstall_test

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	npminstall "github.com/paketo-buildpacks/npm-install"
	"github.com/paketo-buildpacks/npm-install/fakes"
	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/paketo-buildpacks/packit/v2/scribe"
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
		executions    []pexec.Execution
		summer        *fakes.Summer
		environment   *fakes.EnvironmentConfig
		buffer        *bytes.Buffer
		commandOutput *bytes.Buffer

		process npminstall.CIBuildProcess
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
		executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
			executions = append(executions, execution)
			return nil
		}

		summer = &fakes.Summer{}
		environment = &fakes.EnvironmentConfig{}

		environment.GetValueCall.Returns.String = "some-val"

		buffer = bytes.NewBuffer(nil)
		commandOutput = bytes.NewBuffer(nil)

		process = npminstall.NewCIBuildProcess(executable, summer, environment, scribe.NewLogger(buffer))
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

				Expect(summer.SumCall.Receives.Paths[0]).To(Equal(filepath.Join(workingDir, "package.json")))
				Expect(summer.SumCall.Receives.Paths[1]).To(Equal(filepath.Join(workingDir, "package-lock.json")))
				Expect(summer.SumCall.Receives.Paths[2]).To(ContainSubstring("executable_response"))

				Expect(run).To(BeFalse())
				Expect(sha).To(BeEmpty())
				lastExecution := executions[len(executions)-1]
				Expect(lastExecution.Args).To(Equal([]string{
					"get",
					"user-agent",
				}))
				Expect(lastExecution.Dir).To(Equal(workingDir))
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

				Expect(summer.SumCall.Receives.Paths[0]).To(Equal(filepath.Join(workingDir, "package.json")))
				Expect(summer.SumCall.Receives.Paths[1]).To(Equal(filepath.Join(workingDir, "package-lock.json")))
				Expect(summer.SumCall.Receives.Paths[2]).To(ContainSubstring("executable_response"))

				Expect(run).To(BeTrue())
				Expect(sha).To(Equal("other-cache-sha"))

				lastExecution := executions[len(executions)-1]
				Expect(lastExecution.Args).To(Equal([]string{
					"get",
					"user-agent",
				}))
				Expect(lastExecution.Dir).To(Equal(workingDir))
			})
		})

		context("when the layer metadata does not have a checksum", func() {
			it.Before(func() {
				summer.SumCall.Returns.String = "other-cache-sha"
			})

			it("returns false", func() {
				run, sha, err := process.ShouldRun(workingDir, nil)
				Expect(err).NotTo(HaveOccurred())

				Expect(summer.SumCall.Receives.Paths[0]).To(Equal(filepath.Join(workingDir, "package.json")))
				Expect(summer.SumCall.Receives.Paths[1]).To(Equal(filepath.Join(workingDir, "package-lock.json")))
				Expect(summer.SumCall.Receives.Paths[2]).To(ContainSubstring("executable_response"))

				Expect(run).To(BeTrue())
				Expect(sha).To(Equal("other-cache-sha"))

				lastExecution := executions[len(executions)-1]
				Expect(lastExecution.Args).To(Equal([]string{
					"get",
					"user-agent",
				}))
				Expect(lastExecution.Dir).To(Equal(workingDir))
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

			context("when npm get user-agent fails to execute", func() {
				it.Before(func() {
					executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
						return errors.New("very bad error")
					}
					process = npminstall.NewCIBuildProcess(executable, summer, environment, scribe.NewLogger(buffer))
				})

				it("fails", func() {
					_, _, err := process.ShouldRun(workingDir, nil)
					Expect(err).To(MatchError(ContainSubstring("very bad error")))
					Expect(err).To(MatchError(ContainSubstring("failed to execute npm get user-agent")))
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
				Env:    append(os.Environ(), "NPM_CONFIG_LOGLEVEL=some-val"),
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
