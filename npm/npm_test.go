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
	spec.Run(t, "Modules", testNPM, spec.Report(report.Terminal{}))
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
		location = filepath.Join("some", "fake", "dir")
		npmCache = filepath.Join(location, modules.CacheDir)
		mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
	})

	it.After(func() {
		mockCtrl.Finish()
	})

	when("installing", func() {
		when("node_modules and npm-cache do not already exist", func() {
			it("should run npm install and npm cache verify if npm version after 5.0.0", func() {
				mockRunner.EXPECT().RunWithOutput("npm", location, false, "install", "--unsafe-perm", "--cache", npmCache)
				mockRunner.EXPECT().Run("npm", location, false, "cache", "verify", "--cache", npmCache)
				mockRunner.EXPECT().RunWithOutput("npm", location, true, "-v").Return("5.0.0", nil)
				Expect(pkgManager.Install("", "", location)).To(Succeed())
			})

			it("should run npm install and skip npm cache verify if npm version before 5.0.0", func() {
				mockRunner.EXPECT().RunWithOutput("npm", location, false, "install", "--unsafe-perm", "--cache", npmCache)
				mockRunner.EXPECT().RunWithOutput("npm", location, true, "-v").Return("4.3.2", nil)
				Expect(pkgManager.Install("", "", location)).To(Succeed())
			})
		})

		when("node_modules and npm-cache already exist", func() {
			it("should run npm install, npm cache verify, and reuse the existing modules + cache", func() {
				modulesLayer, err := ioutil.TempDir("", "")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(modulesLayer)

				cacheLayer, err := ioutil.TempDir("", "")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(cacheLayer)

				Expect(os.MkdirAll(filepath.Join(modulesLayer, modules.ModulesDir), os.ModePerm)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(modulesLayer, modules.ModulesDir, "module"), []byte(""), os.ModePerm)).To(Succeed())

				Expect(os.MkdirAll(filepath.Join(cacheLayer, modules.CacheDir), os.ModePerm)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(cacheLayer, modules.CacheDir, "cache-item"), []byte(""), os.ModePerm)).To(Succeed())

				npmCache := filepath.Join(location, modules.CacheDir)
				mockRunner.EXPECT().RunWithOutput("npm", location, false, "install", "--unsafe-perm", "--cache", npmCache)
				mockRunner.EXPECT().Run("npm", location, false, "cache", "verify", "--cache", npmCache)
				mockRunner.EXPECT().RunWithOutput("npm", location, true, "-v").Return("5.0.1", nil)

				Expect(pkgManager.Install(modulesLayer, cacheLayer, location)).To(Succeed())

				Expect(filepath.Join(location, modules.ModulesDir, "module")).To(BeARegularFile())
				Expect(filepath.Join(modulesLayer, modules.ModulesDir, "module")).NotTo(BeARegularFile())

				Expect(filepath.Join(location, modules.CacheDir, "cache-item")).To(BeARegularFile())
				Expect(filepath.Join(cacheLayer, modules.CacheDir, "cache-item")).NotTo(BeARegularFile())
			})
		})
	})

	when("rebuilding", func() {
		it("should run npm rebuild", func() {
			cacheLayer, err := ioutil.TempDir("", "")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(cacheLayer)

			mockRunner.EXPECT().Run("npm", location, false, "rebuild")
			mockRunner.EXPECT().RunWithOutput("npm", location, false, "install", "--unsafe-perm", "--cache", npmCache, "--no-audit")

			Expect(pkgManager.Rebuild(cacheLayer, location)).To(Succeed())
		})

	})

	when("Not fully vendored", func() {
		it("warns that unmet dependencies may cause issues", func() {
			modulesLayer, err := ioutil.TempDir("", "")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(modulesLayer)

			cacheLayer, err := ioutil.TempDir("", "")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(cacheLayer)

			debugBuff := bytes.Buffer{}
			infoBuff := bytes.Buffer{}
			npmLogger := logger.Logger{Logger: logger2.NewLogger(&debugBuff, &infoBuff)}
			pkgManager.Logger = npmLogger

			mockRunner.EXPECT().RunWithOutput("npm", location, false, "install", "--unsafe-perm", "--cache", npmCache).Return("unmet peer dependency", nil)
			mockRunner.EXPECT().RunWithOutput("npm", location, true, "-v").Return("4.3.2", nil)

			Expect(pkgManager.Install(modulesLayer, cacheLayer, location)).To(Succeed())
			Expect(infoBuff.String()).To(ContainSubstring(npm.UNMET_DEP_WARNING))
		})
	})
}
