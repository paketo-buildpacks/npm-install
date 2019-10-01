package npm_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	logger2 "github.com/buildpack/libbuildpack/logger"
	"github.com/cloudfoundry/libcfbuildpack/logger"

	"github.com/cloudfoundry/npm-cnb/modules"
	"github.com/cloudfoundry/npm-cnb/npm"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

//go:generate mockgen -source=npm.go -destination=mocks_test.go -package=npm_test

func TestUnitNPM(t *testing.T) {
	spec.Run(t, "NPM", testNPM, spec.Report(report.Terminal{}))
}

func testNPM(t *testing.T, when spec.G, it spec.S) {
	var (
		mockCtrl   *gomock.Controller
		mockRunner *MockRunner
		mockLogger *MockLogger
		pkgManager npm.NPM
		location   string
		npmCache   string
	)

	it.Before(func() {
		RegisterTestingT(t)
		mockCtrl = gomock.NewController(t)
		mockRunner = NewMockRunner(mockCtrl)
		mockLogger = NewMockLogger(mockCtrl)

		pkgManager = npm.NPM{Runner: mockRunner, Logger: mockLogger}

		var err error
		location, err = ioutil.TempDir("", "application")
		Expect(err).NotTo(HaveOccurred())

		npmCache = filepath.Join(location, modules.CacheDir)
		mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
	})

	it.After(func() {
		mockCtrl.Finish()
	})

	when("Install", func() {
		when("node_modules and npm-cache do not already exist", func() {
			when("the npm version is after 5.0.0", func() {
				it.Before(func() {
					mockRunner.EXPECT().RunWithOutput("npm", location, true, "-v").Return("5.0.0", nil)
				})

				it("should run npm install and npm cache verify if npm version after 5.0.0", func() {
					mockRunner.EXPECT().Run("npm", location, false, "install", "--unsafe-perm", "--cache", npmCache)
					mockRunner.EXPECT().Run("npm", location, false, "cache", "verify", "--cache", npmCache)

					Expect(pkgManager.Install("", "", location)).To(Succeed())
				})
			})

			when("the npm version is before 5.0.0", func() {
				it.Before(func() {
					mockRunner.EXPECT().RunWithOutput("npm", location, true, "-v").Return("4.3.2", nil)
				})

				it("should run npm install and skip npm cache verify if npm version before 5.0.0", func() {
					mockRunner.EXPECT().Run("npm", location, false, "install", "--unsafe-perm", "--cache", npmCache)
					Expect(pkgManager.Install("", "", location)).To(Succeed())
				})
			})
		})

		when("node_modules and npm-cache already exist", func() {
			var cacheLayer, modulesLayer string

			it.Before(func() {
				var err error
				modulesLayer, err = ioutil.TempDir("", "modules")
				Expect(err).NotTo(HaveOccurred())

				cacheLayer, err = ioutil.TempDir("", "cache")
				Expect(err).NotTo(HaveOccurred())

				Expect(os.MkdirAll(filepath.Join(modulesLayer, modules.ModulesDir), os.ModePerm)).To(Succeed())

				Expect(ioutil.WriteFile(filepath.Join(modulesLayer, modules.ModulesDir, "module"), []byte{}, os.ModePerm)).To(Succeed())

				Expect(os.MkdirAll(filepath.Join(cacheLayer, modules.CacheDir), os.ModePerm)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(cacheLayer, modules.CacheDir, "cache-item"), []byte{}, os.ModePerm)).To(Succeed())
			})

			it.After(func() {
				Expect(os.RemoveAll(modulesLayer)).To(Succeed())
				Expect(os.RemoveAll(cacheLayer)).To(Succeed())
			})

			it("should run npm install, npm cache verify, and reuse the existing modules + cache", func() {
				npmCache := filepath.Join(location, modules.CacheDir)
				mockRunner.EXPECT().Run("npm", location, false, "install", "--unsafe-perm", "--cache", npmCache)
				mockRunner.EXPECT().Run("npm", location, false, "cache", "verify", "--cache", npmCache)
				mockRunner.EXPECT().RunWithOutput("npm", location, true, "-v").Return("5.0.1", nil)

				Expect(pkgManager.Install(modulesLayer, cacheLayer, location)).To(Succeed())

				Expect(filepath.Join(location, modules.ModulesDir, "module")).To(BeAnExistingFile())
				Expect(filepath.Join(modulesLayer, modules.ModulesDir, "module")).NotTo(BeAnExistingFile())

				Expect(filepath.Join(location, modules.CacheDir, "cache-item")).To(BeAnExistingFile())
				Expect(filepath.Join(cacheLayer, modules.CacheDir, "cache-item")).NotTo(BeAnExistingFile())
			})
		})
	})

	when("CI", func() {
		when("node_modules and npm-cache do not already exist", func() {
			when("the npm version is after 5.0.0", func() {
				it.Before(func() {
					mockRunner.EXPECT().RunWithOutput("npm", location, true, "-v").Return("5.0.0", nil)
				})

				it("should run npm ci and npm cache verify if npm version after 5.0.0", func() {
					mockRunner.EXPECT().Run("npm", location, false, "ci", "--unsafe-perm", "--cache", npmCache)
					mockRunner.EXPECT().Run("npm", location, false, "cache", "verify", "--cache", npmCache)

					Expect(pkgManager.CI("", "", location)).To(Succeed())
				})
			})

			when("the npm version is before 5.0.0", func() {
				it.Before(func() {
					mockRunner.EXPECT().RunWithOutput("npm", location, true, "-v").Return("4.3.2", nil)
				})

				it("should run npm ci and skip npm cache verify if npm version before 5.0.0", func() {
					mockRunner.EXPECT().Run("npm", location, false, "ci", "--unsafe-perm", "--cache", npmCache)
					Expect(pkgManager.CI("", "", location)).To(Succeed())
				})
			})
		})

		when("node_modules and npm-cache already exist", func() {
			var cacheLayer, modulesLayer string

			it.Before(func() {
				var err error
				modulesLayer, err = ioutil.TempDir("", "modules")
				Expect(err).NotTo(HaveOccurred())

				cacheLayer, err = ioutil.TempDir("", "cache")
				Expect(err).NotTo(HaveOccurred())

				Expect(os.MkdirAll(filepath.Join(modulesLayer, modules.ModulesDir), os.ModePerm)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(modulesLayer, modules.ModulesDir, "module"), []byte(""), os.ModePerm)).To(Succeed())

				Expect(os.MkdirAll(filepath.Join(cacheLayer, modules.CacheDir), os.ModePerm)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(cacheLayer, modules.CacheDir, "cache-item"), []byte(""), os.ModePerm)).To(Succeed())
			})

			it.After(func() {
				Expect(os.RemoveAll(modulesLayer)).To(Succeed())
				Expect(os.RemoveAll(cacheLayer)).To(Succeed())
			})

			it("should run npm ci, npm cache verify, and reuse the existing modules + cache", func() {
				npmCache := filepath.Join(location, modules.CacheDir)
				mockRunner.EXPECT().Run("npm", location, false, "ci", "--unsafe-perm", "--cache", npmCache)
				mockRunner.EXPECT().Run("npm", location, false, "cache", "verify", "--cache", npmCache)
				mockRunner.EXPECT().RunWithOutput("npm", location, true, "-v").Return("5.0.1", nil)

				Expect(pkgManager.CI(modulesLayer, cacheLayer, location)).To(Succeed())

				Expect(filepath.Join(location, modules.ModulesDir, "module")).To(BeARegularFile())
				Expect(filepath.Join(modulesLayer, modules.ModulesDir, "module")).NotTo(BeARegularFile())

				Expect(filepath.Join(location, modules.CacheDir, "cache-item")).To(BeARegularFile())
				Expect(filepath.Join(cacheLayer, modules.CacheDir, "cache-item")).NotTo(BeARegularFile())
			})
		})
	})

	when("Rebuild", func() {
		var cacheLayer string

		it.Before(func() {
			var err error
			cacheLayer, err = ioutil.TempDir("", "")
			Expect(err).ToNot(HaveOccurred())
		})

		it.After(func() {
			Expect(os.RemoveAll(cacheLayer)).To(Succeed())
		})

		it("should run npm rebuild", func() {
			mockRunner.EXPECT().Run("npm", location, false, "rebuild")
			mockRunner.EXPECT().Run("npm", location, false, "install", "--unsafe-perm", "--cache", npmCache, "--no-audit")

			Expect(pkgManager.Rebuild(cacheLayer, location)).To(Succeed())
		})
	})

	when("WarnUnmetDependencies", func() {
		it("warns that unmet dependencies may cause issues", func() {
			debugBuff := bytes.Buffer{}
			infoBuff := bytes.Buffer{}
			npmLogger := logger.Logger{Logger: logger2.NewLogger(&debugBuff, &infoBuff)}
			pkgManager.Logger = npmLogger

			mockRunner.EXPECT().RunWithOutput("npm", location, true, "ls").Return("unmet peer dependency", nil)

			Expect(pkgManager.WarnUnmetDependencies(location)).To(Succeed())
			Expect(infoBuff.String()).To(ContainSubstring(npm.UnmetDepWarning))
		})
	})
}
