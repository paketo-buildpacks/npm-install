package npm_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/npm-cnb/npm"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

//go:generate mockgen -source=npm.go -destination=mocks_test.go -package=npm_test

func TestUnitNPM(t *testing.T) {
	RegisterTestingT(t)
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
		mockCtrl = gomock.NewController(t)
		mockRunner = NewMockRunner(mockCtrl)
		mockLogger = NewMockLogger(mockCtrl)
		pkgManager = npm.NPM{Runner: mockRunner, Logger: mockLogger}
	})

	it.After(func() {
		mockCtrl.Finish()
	})

	when("installing", func() {
		when("node_modules and npm-cache do not already exist", func() {
			it("should run npm install and npm cache verify", func() {
				location := filepath.Join("some", "fake", "dir")

				mockLogger.EXPECT().Info("Reusing existing node_modules").Times(0)
				mockLogger.EXPECT().Info("Reusing existing npm-cache").Times(0)

				npmCache := filepath.Join(location, "npm-cache")
				mockRunner.EXPECT().Run("npm", location, "install", "--unsafe-perm", "--cache", npmCache)
				mockRunner.EXPECT().Run("npm", location, "cache", "verify", "--cache", npmCache)

				Expect(pkgManager.Install("", location)).To(Succeed())
			})
		})

		when("node_modules and npm-cache already exist", func() {
			it("should run npm install, npm cache verify, and reuse the existing modules + cache", func() {
				cache, err := ioutil.TempDir("", "")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(cache)

				location, err := ioutil.TempDir("", "")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(location)

				Expect(os.MkdirAll(filepath.Join(cache, "node_modules"), os.ModePerm)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(cache, "node_modules", "module"), []byte(""), os.ModePerm)).To(Succeed())

				Expect(os.MkdirAll(filepath.Join(cache, "npm-cache"), os.ModePerm)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(cache, "npm-cache", "cache-item"), []byte(""), os.ModePerm)).To(Succeed())

				mockLogger.EXPECT().Info("Reusing existing node_modules")
				mockLogger.EXPECT().Info("Reusing existing npm-cache")

				npmCache := filepath.Join(location, "npm-cache")
				mockRunner.EXPECT().Run("npm", location, "install", "--unsafe-perm", "--cache", npmCache)
				mockRunner.EXPECT().Run("npm", location, "cache", "verify", "--cache", npmCache)

				Expect(pkgManager.Install(cache, location)).To(Succeed())
			})
		})
	})

	when("rebuilding", func() {
		it("should run npm rebuild", func() {
			location := filepath.Join("some", "fake", "dir")

			mockRunner.EXPECT().Run("npm", location, "rebuild")

			Expect(pkgManager.Rebuild(location)).To(Succeed())
		})
	})
}
