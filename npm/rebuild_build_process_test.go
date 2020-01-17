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
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testRebuildBuildProcess(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layerDir   string
		cacheDir   string
		workingDir string

		scriptsParser *fakes.ScriptsParser
		executable    *fakes.Executable

		process npm.RebuildBuildProcess
	)

	context("Run", func() {
		var executions []pexec.Execution

		it.Before(func() {
			var err error
			layerDir, err = ioutil.TempDir("", "layer")
			Expect(err).NotTo(HaveOccurred())

			cacheDir, err = ioutil.TempDir("", "layer")
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

			process = npm.NewRebuildBuildProcess(executable, scriptsParser)
		})

		it.After(func() {
			Expect(os.RemoveAll(layerDir)).To(Succeed())
			Expect(os.RemoveAll(cacheDir)).To(Succeed())
			Expect(os.RemoveAll(workingDir)).To(Succeed())
		})

		it("runs the npm rebuild command", func() {
			Expect(process.Run(layerDir, cacheDir, workingDir)).To(Succeed())

			Expect(executable.ExecuteCall.CallCount).To(Equal(2))
			Expect(executions).To(Equal([]pexec.Execution{
				{
					Args:   []string{"list"},
					Dir:    workingDir,
					Stdout: bytes.NewBuffer(nil),
					Stderr: bytes.NewBuffer(nil),
				},
				{
					Args: []string{"rebuild", fmt.Sprintf("--nodedir=")},
					Dir:  workingDir,
				},
			}))

			path, err := os.Readlink(filepath.Join(workingDir, "node_modules"))
			Expect(err).NotTo(HaveOccurred())
			Expect(path).To(Equal(filepath.Join(layerDir, "node_modules")))
		})

		context("when the package.json includes preinstall and postinstall scripts", func() {
			it.Before(func() {
				scriptsParser.ParseScriptsCall.Returns.Scripts = map[string]string{
					"preinstall":  "some-preinstall-script",
					"postinstall": "some-postinstall-script",
				}
			})

			it("runs the scripts before and after it runs the npm rebuild command", func() {
				Expect(process.Run(layerDir, cacheDir, workingDir)).To(Succeed())

				Expect(executable.ExecuteCall.CallCount).To(Equal(4))
				Expect(executions).To(Equal([]pexec.Execution{
					{
						Args:   []string{"list"},
						Dir:    workingDir,
						Stdout: bytes.NewBuffer(nil),
						Stderr: bytes.NewBuffer(nil),
					},
					{
						Args: []string{"run-script", "preinstall"},
						Dir:  workingDir,
					},
					{
						Args: []string{"rebuild", fmt.Sprintf("--nodedir=")},
						Dir:  workingDir,
					},
					{
						Args: []string{"run-script", "postinstall"},
						Dir:  workingDir,
					},
				}))

				path, err := os.Readlink(filepath.Join(workingDir, "node_modules"))
				Expect(err).NotTo(HaveOccurred())
				Expect(path).To(Equal(filepath.Join(layerDir, "node_modules")))
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

					err := process.Run(layerDir, cacheDir, workingDir)
					Expect(err).To(MatchError("vendored node_modules have unmet dependencies:\nstdout output\nstderr output\n\nexit status 1"))
				})
			})

			context("when the node_modules directory cannot be created", func() {
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

			context("parsing package.json for scripts", func() {
				it.Before(func() {
					scriptsParser.ParseScriptsCall.Returns.Err = errors.New("a parsing error")
				})

				it("returns an error", func() {
					err := process.Run(layerDir, cacheDir, workingDir)
					Expect(err).To(MatchError("failed to parse package.json: a parsing error"))
				})
			})

			context("when preinstall scripts fail", func() {
				it.Before(func() {
					scriptsParser.ParseScriptsCall.Returns.Scripts = map[string]string{"preinstall": "some pre-install scripts"}

					executable.ExecuteCall.Stub = func(execution pexec.Execution) (string, string, error) {
						if strings.Contains(strings.Join(execution.Args, " "), "preinstall") {
							return "", "", fmt.Errorf("an actual error")
						}

						return "", "", nil
					}
				})

				it("returns an error", func() {
					err := process.Run(layerDir, cacheDir, workingDir)
					Expect(err).To(MatchError("preinstall script failed on rebuild: an actual error"))
				})
			})

			context("when the executable fails to run rebuild", func() {
				it.Before(func() {
					executable.ExecuteCall.Stub = func(execution pexec.Execution) (string, string, error) {
						if strings.Contains(strings.Join(execution.Args, " "), "rebuild") {
							return "", "", errors.New("failed to rebuild")
						}

						return "", "", nil
					}
				})

				it("returns an error", func() {
					err := process.Run(layerDir, cacheDir, workingDir)
					Expect(err).To(MatchError("npm rebuild failed: failed to rebuild"))
				})
			})

			context("when postinstall scripts fail", func() {
				it.Before(func() {
					scriptsParser.ParseScriptsCall.Returns.Scripts = map[string]string{"postinstall": "some post-install scripts"}

					executable.ExecuteCall.Stub = func(execution pexec.Execution) (string, string, error) {
						if strings.Contains(strings.Join(execution.Args, " "), "postinstall") {
							return "", "", fmt.Errorf("an actual error")
						}

						return "", "", nil
					}
				})

				it("returns an error", func() {
					err := process.Run(layerDir, cacheDir, workingDir)
					Expect(err).To(MatchError("postinstall script failed on rebuild: an actual error"))
				})
			})
		})
	})
}
