package npm_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

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
	)

	it.Before(func() {
		RegisterTestingT(t)
		mockCtrl = gomock.NewController(t)
		mockRunner = NewMockRunner(mockCtrl)
		mockLogger = NewMockLogger(mockCtrl)

		mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

		pkgManager = npm.NPM{Runner: mockRunner, Logger: mockLogger}
	})

	it.After(func() {
		mockCtrl.Finish()
	})

	when("installing", func() {
		when("node_modules and npm-cache do not already exist", func() {
			it("should run npm install and npm cache verify if npm version after 5.0.0", func() {
				location := filepath.Join("some", "fake", "dir")

				npmCache := filepath.Join(location, modules.CacheDir)
				mockRunner.EXPECT().Run("npm", location, "install", "--unsafe-perm", "--cache", npmCache)
				mockRunner.EXPECT().Run("npm", location, "cache", "verify", "--cache", npmCache)
				mockRunner.EXPECT().RunWithOutput("npm", location, "-v").Return("5.0.0", nil)
				Expect(pkgManager.Install("", "", location)).To(Succeed())
			})

			it("should run npm install and skip npm cache verify if npm version before 5.0.0", func() {
				location := filepath.Join("some", "fake", "dir")

				npmCache := filepath.Join(location, modules.CacheDir)
				mockRunner.EXPECT().Run("npm", location, "install", "--unsafe-perm", "--cache", npmCache)
				mockRunner.EXPECT().RunWithOutput("npm", location, "-v").Return("4.3.2", nil)
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

				location, err := ioutil.TempDir("", "")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(location)

				Expect(os.MkdirAll(filepath.Join(modulesLayer, modules.ModulesDir), os.ModePerm)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(modulesLayer, modules.ModulesDir, "module"), []byte(""), os.ModePerm)).To(Succeed())

				Expect(os.MkdirAll(filepath.Join(cacheLayer, modules.CacheDir), os.ModePerm)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(cacheLayer, modules.CacheDir, "cache-item"), []byte(""), os.ModePerm)).To(Succeed())

				npmCache := filepath.Join(location, modules.CacheDir)
				mockRunner.EXPECT().Run("npm", location, "install", "--unsafe-perm", "--cache", npmCache)
				mockRunner.EXPECT().Run("npm", location, "cache", "verify", "--cache", npmCache)
				mockRunner.EXPECT().RunWithOutput("npm", location, "-v").Return("5.0.1", nil)

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
			location, err := ioutil.TempDir("", "")
			Expect(err).ToNot(HaveOccurred())

			npmCache := filepath.Join(location, modules.CacheDir)
			cacheLayer, err := ioutil.TempDir("", "")
			Expect(err).ToNot(HaveOccurred())

			mockRunner.EXPECT().Run("npm", location, "rebuild")
			mockRunner.EXPECT().Run("npm", location, "install", "--unsafe-perm", "--cache", npmCache)

			Expect(pkgManager.Rebuild(cacheLayer, location)).To(Succeed())
		})
	})
}
