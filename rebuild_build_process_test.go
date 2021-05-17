package npminstall_test

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	npminstall "github.com/paketo-buildpacks/npm-install"
	"github.com/paketo-buildpacks/npm-install/fakes"
	"github.com/paketo-buildpacks/packit/pexec"
	"github.com/paketo-buildpacks/packit/scribe"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testRebuildBuildProcess(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		executions []pexec.Execution

		modulesDir string
		cacheDir   string
		workingDir string

		executable  *fakes.Executable
		summer      *fakes.Summer
		environment *fakes.EnvironmentConfig

		buffer        *bytes.Buffer
		commandOutput *bytes.Buffer

		process npminstall.RebuildBuildProcess
	)

	it.Before(func() {
		var err error
		modulesDir, err = ioutil.TempDir("", "modules")
		Expect(err).NotTo(HaveOccurred())

		cacheDir, err = ioutil.TempDir("", "cache")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		Expect(os.MkdirAll(filepath.Join(workingDir, "node_modules", "some-module"), os.ModePerm)).To(Succeed())

		err = ioutil.WriteFile(filepath.Join(workingDir, "node_modules", "some-module", "some-file"), []byte("some-content"), 0644)
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

		process = npminstall.NewRebuildBuildProcess(executable, summer, environment, scribe.NewLogger(buffer))
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
				Expect(summer.SumCall.Receives.Paths[0]).To(Equal(filepath.Join(workingDir, "node_modules")))
				Expect(summer.SumCall.Receives.Paths[1]).To(ContainSubstring("executable_response"))
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
				Expect(run).To(BeTrue())
				Expect(sha).To(Equal("other-cache-sha"))
				Expect(summer.SumCall.Receives.Paths[0]).To(Equal(filepath.Join(workingDir, "node_modules")))
				Expect(summer.SumCall.Receives.Paths[1]).To(ContainSubstring("executable_response"))
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
				Expect(run).To(BeTrue())
				Expect(sha).To(Equal("other-cache-sha"))

				Expect(summer.SumCall.Receives.Paths[0]).To(Equal(filepath.Join(workingDir, "node_modules")))
				Expect(summer.SumCall.Receives.Paths[1]).To(ContainSubstring("executable_response"))
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
					process = npminstall.NewRebuildBuildProcess(executable, summer, environment, scribe.NewLogger(buffer))
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
		it("runs the npm rebuild command", func() {
			Expect(process.Run(modulesDir, cacheDir, workingDir)).To(Succeed())

			Expect(executable.ExecuteCall.CallCount).To(Equal(4))
			Expect(executions).To(Equal([]pexec.Execution{
				{
					Args:   []string{"list"},
					Dir:    workingDir,
					Stdout: commandOutput,
					Stderr: commandOutput,
				},
				{
					Args:   []string{"run-script", "preinstall", "--if-present"},
					Dir:    workingDir,
					Stdout: commandOutput,
					Stderr: commandOutput,
				},
				{
					Args:   []string{"rebuild", "--nodedir="},
					Dir:    workingDir,
					Env:    append(os.Environ(), "NPM_CONFIG_LOGLEVEL=some-val"),
					Stdout: commandOutput,
					Stderr: commandOutput,
				},
				{
					Args:   []string{"run-script", "postinstall", "--if-present"},
					Dir:    workingDir,
					Stdout: commandOutput,
					Stderr: commandOutput,
				},
			}))

			path, err := os.Readlink(filepath.Join(workingDir, "node_modules"))
			Expect(err).NotTo(HaveOccurred())
			Expect(path).To(Equal(filepath.Join(modulesDir, "node_modules")))
		})

		context("when the package.json includes preinstall and postinstall scripts", func() {
			it("runs the scripts before and after it runs the npm rebuild command", func() {
				Expect(process.Run(modulesDir, cacheDir, workingDir)).To(Succeed())

				Expect(executable.ExecuteCall.CallCount).To(Equal(4))
				Expect(executions).To(Equal([]pexec.Execution{
					{
						Args:   []string{"list"},
						Dir:    workingDir,
						Stdout: commandOutput,
						Stderr: commandOutput,
					},
					{
						Args:   []string{"run-script", "preinstall", "--if-present"},
						Dir:    workingDir,
						Stdout: commandOutput,
						Stderr: commandOutput,
					},
					{
						Args:   []string{"rebuild", "--nodedir="},
						Dir:    workingDir,
						Env:    append(os.Environ(), "NPM_CONFIG_LOGLEVEL=some-val"),
						Stdout: commandOutput,
						Stderr: commandOutput,
					},
					{
						Args:   []string{"run-script", "postinstall", "--if-present"},
						Dir:    workingDir,
						Stdout: commandOutput,
						Stderr: commandOutput,
					},
				}))

				path, err := os.Readlink(filepath.Join(workingDir, "node_modules"))
				Expect(err).NotTo(HaveOccurred())
				Expect(path).To(Equal(filepath.Join(modulesDir, "node_modules")))
			})
		})

		context("failure cases", func() {
			context("when npm list fails", func() {
				it("returns an error", func() {
					executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
						if strings.Contains(strings.Join(execution.Args, " "), "list") {
							fmt.Fprintln(execution.Stdout, "stdout output")
							fmt.Fprintln(execution.Stderr, "stderr output")
							return errors.New("exit status 1")
						}

						return nil
					}

					err := process.Run(modulesDir, cacheDir, workingDir)
					Expect(buffer.String()).To(ContainSubstring("    stdout output\n    stderr output\n"))
					Expect(err).To(MatchError("vendored node_modules have unmet dependencies: npm list failed: exit status 1"))
				})
			})

			context("when the node_modules directory cannot be created", func() {
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

			context("when preinstall scripts fail", func() {
				it.Before(func() {
					executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
						if strings.Contains(strings.Join(execution.Args, " "), "preinstall") {
							fmt.Fprintln(execution.Stderr, "pre-install on stdout")
							fmt.Fprintln(execution.Stdout, "pre-install on stderr")
							return fmt.Errorf("an actual error")
						}

						return nil
					}
				})

				it("returns an error", func() {
					err := process.Run(modulesDir, cacheDir, workingDir)
					Expect(buffer.String()).To(ContainSubstring("    pre-install on stdout\n    pre-install on stderr\n"))
					Expect(err).To(MatchError("preinstall script failed on rebuild: an actual error"))
				})
			})

			context("when the executable fails to run rebuild", func() {
				it.Before(func() {
					executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
						if strings.Contains(strings.Join(execution.Args, " "), "rebuild") {
							fmt.Fprintln(execution.Stderr, "rebuild error on stdout")
							fmt.Fprintln(execution.Stdout, "rebuild error on stderr")
							return errors.New("failed to rebuild")
						}

						return nil
					}
				})

				it("returns an error", func() {
					err := process.Run(modulesDir, cacheDir, workingDir)
					Expect(buffer.String()).To(ContainSubstring("    rebuild error on stdout\n    rebuild error on stderr\n"))
					Expect(err).To(MatchError("npm rebuild failed: failed to rebuild"))
				})
			})

			context("when postinstall scripts fail", func() {
				it.Before(func() {
					executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
						if strings.Contains(strings.Join(execution.Args, " "), "postinstall") {
							fmt.Fprintln(execution.Stderr, "postinstall on stdout")
							fmt.Fprintln(execution.Stdout, "postinstall on stderr")
							return fmt.Errorf("an actual error")
						}

						return nil
					}
				})

				it("returns an error", func() {
					err := process.Run(modulesDir, cacheDir, workingDir)
					Expect(buffer.String()).To(ContainSubstring("    postinstall on stdout\n    postinstall on stderr\n"))
					Expect(err).To(MatchError("postinstall script failed on rebuild: an actual error"))
				})
			})
		})
	})
}
