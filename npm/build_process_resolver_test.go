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

func testBuildProcessResolver(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layerDir   string
		workingDir string
		executable *fakes.Executable
		resolver   npm.BuildProcessResolver
	)

	it.Before(func() {
		var err error
		layerDir, err = ioutil.TempDir("", "layer")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		executable = &fakes.Executable{}

		resolver = npm.NewBuildProcessResolver(executable)
	})

	it.After(func() {
		Expect(os.RemoveAll(layerDir)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("Resolve", func() {
		context("when the node_modules directory does not exist in the working directory", func() {
			var process npm.BuildProcess

			it.Before(func() {
				var err error
				process, err = resolver.Resolve(workingDir)
				Expect(err).NotTo(HaveOccurred())
			})

			it("resolves a process that installs the node modules", func() {
				err := process(layerDir, workingDir)
				Expect(err).NotTo(HaveOccurred())

				Expect(executable.ExecuteCall.Receives.Execution).To(Equal(pexec.Execution{
					Args: []string{"install"},
					Dir:  workingDir,
				}))

				path, err := os.Readlink(filepath.Join(workingDir, "node_modules"))
				Expect(err).NotTo(HaveOccurred())
				Expect(path).To(Equal(filepath.Join(layerDir, "node_modules")))
			})

			context("failure cases", func() {
				context("when the node_modules directory cannot be created", func() {
					it.Before(func() {
						Expect(os.Chmod(layerDir, 0000)).To(Succeed())
					})

					it("returns an error", func() {
						err := process(layerDir, workingDir)
						Expect(err).To(MatchError(ContainSubstring("permission denied")))
					})
				})

				context("when the node_modules directory cannot be symlinked into the working directory", func() {
					it.Before(func() {
						Expect(os.Chmod(workingDir, 0000)).To(Succeed())
					})

					it("returns an error", func() {
						err := process(layerDir, workingDir)
						Expect(err).To(MatchError(ContainSubstring("permission denied")))
					})
				})

				context("when the executable fails", func() {
					it.Before(func() {
						executable.ExecuteCall.Returns.Err = errors.New("failed to execute")
					})

					it("returns an error", func() {
						err := process(layerDir, workingDir)
						Expect(err).To(MatchError("failed to execute"))
					})
				})
			})
		})

		context("when the node_modules directory exists in the working directory", func() {
			var (
				process    npm.BuildProcess
				executions []pexec.Execution
			)

			it.Before(func() {
				Expect(os.MkdirAll(filepath.Join(workingDir, "node_modules", "some-module"), os.ModePerm)).To(Succeed())

				err := ioutil.WriteFile(filepath.Join(workingDir, "node_modules", "some-module", "some-file"), []byte("some-content"), 0644)
				Expect(err).NotTo(HaveOccurred())

				process, err = resolver.Resolve(workingDir)
				Expect(err).NotTo(HaveOccurred())

				executable.ExecuteCall.Stub = func(execution pexec.Execution) (string, string, error) {
					executions = append(executions, execution)
					return "", "", nil
				}
			})

			it("rebuilds the node modules", func() {
				err := process(layerDir, workingDir)
				Expect(err).NotTo(HaveOccurred())

				Expect(executable.ExecuteCall.CallCount).To(Equal(2))
				Expect(executions[0]).To(Equal(pexec.Execution{
					Args: []string{"rebuild"},
					Dir:  workingDir,
				}))
				Expect(executions[1]).To(Equal(pexec.Execution{
					Args: []string{"install"},
					Dir:  workingDir,
				}))

				path, err := os.Readlink(filepath.Join(workingDir, "node_modules"))
				Expect(err).NotTo(HaveOccurred())
				Expect(path).To(Equal(filepath.Join(layerDir, "node_modules")))

				contents, err := ioutil.ReadFile(filepath.Join(layerDir, "node_modules", "some-module", "some-file"))
				Expect(err).NotTo(HaveOccurred())
				Expect(string(contents)).To(Equal("some-content"))
			})

			context("failure cases", func() {
				context("when the working directory is unreadable", func() {
					it.Before(func() {
						Expect(os.Chmod(workingDir, 0000)).To(Succeed())
					})

					it.After(func() {
						Expect(os.Chmod(workingDir, os.ModePerm)).To(Succeed())
					})

					it("returns an error", func() {
						_, err := resolver.Resolve(workingDir)
						Expect(err).To(MatchError(ContainSubstring("permission denied")))
					})
				})

				context("when the node_modules directory cannot be created", func() {
					it.Before(func() {
						Expect(os.Chmod(layerDir, 0000)).To(Succeed())
					})

					it("returns an error", func() {
						err := process(layerDir, workingDir)
						Expect(err).To(MatchError(ContainSubstring("permission denied")))
					})
				})

				context("when the executable fails to run rebuild", func() {
					it.Before(func() {
						executable.ExecuteCall.Stub = func(pexec.Execution) (string, string, error) {
							if executable.ExecuteCall.CallCount == 1 {
								return "", "", errors.New("failed to rebuild")
							}

							return "", "", nil
						}
					})

					it("returns an error", func() {
						err := process(layerDir, workingDir)
						Expect(err).To(MatchError("failed to rebuild"))
					})
				})

				context("when the executable fails to run install", func() {
					it.Before(func() {
						executable.ExecuteCall.Stub = func(pexec.Execution) (string, string, error) {
							if executable.ExecuteCall.CallCount == 2 {
								return "", "", errors.New("failed to install")
							}

							return "", "", nil
						}
					})

					it("returns an error", func() {
						err := process(layerDir, workingDir)
						Expect(err).To(MatchError("failed to install"))
					})
				})
			})
		})
	})
}
