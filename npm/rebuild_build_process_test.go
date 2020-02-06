package npm_test

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudfoundry/npm-cnb/npm"
	"github.com/cloudfoundry/npm-cnb/npm/fakes"
	"github.com/cloudfoundry/packit/pexec"
	"github.com/cloudfoundry/packit/scribe"
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

		scriptsParser *fakes.ScriptsParser
		executable    *fakes.Executable
		summer        *fakes.Summer

		buffer *bytes.Buffer

		process npm.RebuildBuildProcess
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
		executable.ExecuteCall.Stub = func(execution pexec.Execution) (string, string, error) {
			executions = append(executions, execution)
			return "", "", nil
		}

		scriptsParser = &fakes.ScriptsParser{}
		summer = &fakes.Summer{}

		buffer = bytes.NewBuffer(nil)
		logger := scribe.NewLogger(buffer)

		process = npm.NewRebuildBuildProcess(executable, scriptsParser, summer, logger)
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

				Expect(summer.SumCall.Receives.Path).To(Equal(filepath.Join(workingDir, "node_modules")))
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

				Expect(summer.SumCall.Receives.Path).To(Equal(filepath.Join(workingDir, "node_modules")))
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

				Expect(summer.SumCall.Receives.Path).To(Equal(filepath.Join(workingDir, "node_modules")))
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

		it("runs the npm rebuild command", func() {
			Expect(process.Run(modulesDir, cacheDir, workingDir)).To(Succeed())

			Expect(executable.ExecuteCall.CallCount).To(Equal(2))
			Expect(executions).To(Equal([]pexec.Execution{
				{
					Args:   []string{"list"},
					Dir:    workingDir,
					Stdout: buffer,
					Stderr: buffer,
				},
				{
					Args:   []string{"rebuild", fmt.Sprintf("--nodedir=")},
					Dir:    workingDir,
					Env:    append(os.Environ(), "NPM_CONFIG_PRODUCTION=true", "NPM_CONFIG_LOGLEVEL=error"),
					Stdout: buffer,
					Stderr: buffer,
				},
			}))

			path, err := os.Readlink(filepath.Join(workingDir, "node_modules"))
			Expect(err).NotTo(HaveOccurred())
			Expect(path).To(Equal(filepath.Join(modulesDir, "node_modules")))
		})

		context("when the package.json includes preinstall and postinstall scripts", func() {
			it.Before(func() {
				scriptsParser.ParseScriptsCall.Returns.Scripts = map[string]string{
					"preinstall":  "some-preinstall-script",
					"postinstall": "some-postinstall-script",
				}
			})

			it("runs the scripts before and after it runs the npm rebuild command", func() {
				Expect(process.Run(modulesDir, cacheDir, workingDir)).To(Succeed())

				Expect(executable.ExecuteCall.CallCount).To(Equal(4))
				Expect(executions).To(Equal([]pexec.Execution{
					{
						Args:   []string{"list"},
						Dir:    workingDir,
						Stdout: buffer,
						Stderr: buffer,
					},
					{
						Args:   []string{"run-script", "preinstall"},
						Dir:    workingDir,
						Stdout: buffer,
						Stderr: buffer,
					},
					{
						Args:   []string{"rebuild", fmt.Sprintf("--nodedir=")},
						Dir:    workingDir,
						Env:    append(os.Environ(), "NPM_CONFIG_PRODUCTION=true", "NPM_CONFIG_LOGLEVEL=error"),
						Stdout: buffer,
						Stderr: buffer,
					},
					{
						Args:   []string{"run-script", "postinstall"},
						Dir:    workingDir,
						Stdout: buffer,
						Stderr: buffer,
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
					executable.ExecuteCall.Stub = func(execution pexec.Execution) (string, string, error) {
						if strings.Contains(strings.Join(execution.Args, " "), "list") {
							fmt.Fprintln(execution.Stdout, "stdout output")
							fmt.Fprintln(execution.Stderr, "stderr output")
							return "", "", errors.New("exit status 1")
						}

						return "", "", nil
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

			context("parsing package.json for scripts", func() {
				it.Before(func() {
					scriptsParser.ParseScriptsCall.Returns.Err = errors.New("a parsing error")
				})

				it("returns an error", func() {
					err := process.Run(modulesDir, cacheDir, workingDir)
					Expect(err).To(MatchError("failed to parse package.json: a parsing error"))
				})
			})

			context("when preinstall scripts fail", func() {
				it.Before(func() {
					scriptsParser.ParseScriptsCall.Returns.Scripts = map[string]string{"preinstall": "some pre-install scripts"}

					executable.ExecuteCall.Stub = func(execution pexec.Execution) (string, string, error) {
						if strings.Contains(strings.Join(execution.Args, " "), "preinstall") {
							fmt.Fprintln(execution.Stderr, "pre-install on stdout")
							fmt.Fprintln(execution.Stdout, "pre-install on stderr")
							return "", "", fmt.Errorf("an actual error")
						}

						return "", "", nil
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
					executable.ExecuteCall.Stub = func(execution pexec.Execution) (string, string, error) {
						if strings.Contains(strings.Join(execution.Args, " "), "rebuild") {
							fmt.Fprintln(execution.Stderr, "rebuild error on stdout")
							fmt.Fprintln(execution.Stdout, "rebuild error on stderr")
							return "", "", errors.New("failed to rebuild")
						}

						return "", "", nil
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
					scriptsParser.ParseScriptsCall.Returns.Scripts = map[string]string{"postinstall": "some post-install scripts"}

					executable.ExecuteCall.Stub = func(execution pexec.Execution) (string, string, error) {
						if strings.Contains(strings.Join(execution.Args, " "), "postinstall") {
							fmt.Fprintln(execution.Stderr, "postinstall on stdout")
							fmt.Fprintln(execution.Stdout, "postinstall on stderr")
							return "", "", fmt.Errorf("an actual error")
						}

						return "", "", nil
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
