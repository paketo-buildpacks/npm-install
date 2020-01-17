package npm_test

import (
	"bytes"
	"fmt"
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

		layerDir      string
		cacheDir      string
		workingDir    string
		scriptsParser *fakes.ScriptsParser
		executable    *fakes.Executable

		solutionsMap map[[3]bool]string
	)
	solutionsMap = map[[3]bool]string{
		// package-lock.json | node_modules | npm-cache => npm command
		[3]bool{false, false, false}: "install",
		[3]bool{false, false, true}:  "install",
		[3]bool{false, true, false}:  "rebuild",
		[3]bool{false, true, true}:   "rebuild",
		[3]bool{true, false, false}:  "ci",
		[3]bool{true, false, true}:   "ci",
		[3]bool{true, true, false}:   "rebuild",
		[3]bool{true, true, true}:    "ci",
	}

	for _, i := range []bool{false, true} {
		for _, j := range []bool{false, true} {
			for _, k := range []bool{false, true} {

				stateArray := [3]bool{i, j, k}
				packageLockExist, nodeModulesExist, npmCacheExist := stateArray[0], stateArray[1], stateArray[2]
				specName := fmt.Sprintf("NodeModules: %v, package-lock.json: %v, npmCache: %v", nodeModulesExist, packageLockExist, npmCacheExist)
				var argsMap map[string][]pexec.Execution
				context(specName, func() {
					var executionCalls []pexec.Execution
					var process npm.BuildProcess
					var resolver npm.BuildProcessResolver
					it.Before(func() {
						var err error
						layerDir, err = ioutil.TempDir("", "layer")
						Expect(err).NotTo(HaveOccurred())

						cacheDir, err = ioutil.TempDir("", "layer")
						Expect(err).NotTo(HaveOccurred())

						workingDir, err = ioutil.TempDir("", "working-dir")
						Expect(err).NotTo(HaveOccurred())

						executable = &fakes.Executable{}
						scriptsParser = &fakes.ScriptsParser{}

						executionCalls = []pexec.Execution{}

						executable.ExecuteCall.Stub = func(param1 pexec.Execution) (string, string, error) {
							executionCalls = append(executionCalls, param1)
							return "", "", nil
						}
						argsMap = map[string][]pexec.Execution{
							"install": {
								{
									Args: []string{"install", "--unsafe-perm", "--cache", cacheDir},
									Dir:  workingDir,
								},
							},
							"rebuild": {
								{
									Args:   []string{"list"},
									Dir:    workingDir,
									Stdout: bytes.NewBuffer(nil),
									Stderr: bytes.NewBuffer(nil),
								},
								{
									Args: []string{"rebuild", fmt.Sprintf("--nodedir=%s", os.Getenv("NODE_HOME"))},
									Dir:  workingDir,
								},
							},
							"ci": {
								{
									Args: []string{"ci", "--unsafe-perm", "--cache", cacheDir},
									Dir:  workingDir,
								},
							},
						}

						resolver = npm.NewBuildProcessResolver(executable, scriptsParser)
					})

					it.After(func() {
						Expect(os.RemoveAll(layerDir)).To(Succeed())
						Expect(os.RemoveAll(workingDir)).To(Succeed())
						Expect(os.RemoveAll(cacheDir)).To(Succeed())
					})

					context("Resolve and run installation process", func() {
						it.Before(func() {
							if nodeModulesExist {
								Expect(os.MkdirAll(filepath.Join(workingDir, "node_modules", "some-module"), os.ModePerm)).To(Succeed())

								err := ioutil.WriteFile(filepath.Join(workingDir, "node_modules", "some-module", "some-file"), []byte("some-content"), 0644)
								Expect(err).NotTo(HaveOccurred())
							}
							if packageLockExist {
								// make packageLock
								Expect(ioutil.WriteFile(filepath.Join(workingDir, "package-lock.json"), []byte(""), os.ModePerm)).To(Succeed())
							}
							if npmCacheExist {
								// make npmCache
								Expect(os.MkdirAll(filepath.Join(workingDir, "npm-cache"), os.ModePerm)).To(Succeed())

								err := ioutil.WriteFile(filepath.Join(workingDir, "npm-cache", "some-cache-file"), []byte("some-content"), 0644)
								Expect(err).NotTo(HaveOccurred())
							}

							var err error
							process, err = resolver.Resolve(workingDir, cacheDir)
							Expect(err).NotTo(HaveOccurred())

						})
						it(fmt.Sprintf("runs npm and succeeds"), func() {
							Expect(process.Run(layerDir, cacheDir, workingDir)).To(Succeed())
							Expect(executionCalls).To(Equal(argsMap[solutionsMap[stateArray]]))

							if nodeModulesExist {
								path, err := os.Readlink(filepath.Join(workingDir, "node_modules"))
								Expect(err).NotTo(HaveOccurred())
								Expect(path).To(Equal(filepath.Join(layerDir, "node_modules")))

								contents, err := ioutil.ReadFile(filepath.Join(layerDir, "node_modules", "some-module", "some-file"))
								Expect(err).NotTo(HaveOccurred())
								Expect(string(contents)).To(Equal("some-content"))
							}

							if npmCacheExist {
								contents, err := ioutil.ReadFile(filepath.Join(cacheDir, "npm-cache", "some-cache-file"))
								Expect(err).NotTo(HaveOccurred())
								Expect(string(contents)).To(Equal("some-content"))
							}
						})
					})
				})
			}
		}
	}

	context("failure cases", func() {
		var (
			resolver   npm.BuildProcessResolver
			executable *fakes.Executable
		)

		it.Before(func() {

			var err error
			layerDir, err = ioutil.TempDir("", "layer")
			Expect(err).NotTo(HaveOccurred())

			cacheDir, err = ioutil.TempDir("", "layer")
			Expect(err).NotTo(HaveOccurred())

			workingDir, err = ioutil.TempDir("", "working-dir")
			Expect(err).NotTo(HaveOccurred())

			executable = &fakes.Executable{}
			scriptsParser = &fakes.ScriptsParser{}

			resolver = npm.NewBuildProcessResolver(executable, scriptsParser)
		})

		it.After(func() {
			Expect(os.RemoveAll(layerDir)).To(Succeed())
			Expect(os.RemoveAll(workingDir)).To(Succeed())
			Expect(os.RemoveAll(cacheDir)).To(Succeed())
		})

		context("Resolve", func() {
			context("when the working directory is unreadable", func() {
				it.Before(func() {
					Expect(os.Chmod(workingDir, 0000)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Chmod(workingDir, os.ModePerm)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := resolver.Resolve(workingDir, cacheDir)
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})

			context("npm-cache exists and is unreadable", func() {
				var npmCacheItemPath string

				it.Before(func() {
					npmCacheItemPath = filepath.Join(workingDir, "npm-cache", "some-cache-dir")
					Expect(os.MkdirAll(npmCacheItemPath, os.ModePerm)).To(Succeed())
					Expect(os.Chmod(npmCacheItemPath, 0000)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Chmod(npmCacheItemPath, os.ModePerm)).To(Succeed())
				})

				it("fails", func() {
					_, err := resolver.Resolve(workingDir, cacheDir)
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})
		})
	})
}
