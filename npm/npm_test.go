package npm_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/libjavabuildpack/test"
	"github.com/cloudfoundry/npm-cnb/detect"
	. "github.com/cloudfoundry/npm-cnb/npm"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

//go:generate mockgen -source=runner.go -destination=mocks_test.go -package=npm_test

func TestUnitNpm(t *testing.T) {
	RegisterTestingT(t)
	spec.Run(t, "Build", testNpm, spec.Report(report.Terminal{}))
}

func testNpm(t *testing.T, when spec.G, it spec.S) {
	var (
		mockCtrl   *gomock.Controller
		mockRunner *MockRunner
		Npm        *NPM
		f          test.BuildFactory
		err        error
		cacheLayer string
		appRoot    string
	)

	it.Before(func() {
		f = test.NewBuildFactory(t)
		mockCtrl = gomock.NewController(t)
		mockRunner = NewMockRunner(mockCtrl)
		Npm = &NPM{Runner: mockRunner}
		cacheLayer = f.Build.Cache.Layer(detect.NPMDependency).Root
		appRoot = f.Build.Application.Root
	})

	it.After(func() {
		mockCtrl.Finish()
	})

	when("RebuildLayer", func() {
		it.Before(func() {
			err = os.MkdirAll(appRoot, 0777)
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(filepath.Join(appRoot, "package.json"), []byte("package json"), 0666)
			Expect(err).To(BeNil())

			err = os.MkdirAll(filepath.Join(appRoot, "node_modules"), 0777)
			Expect(err).To(BeNil())

			err = os.MkdirAll(f.Build.Cache.Root, 0777)
			Expect(err).To(BeNil())
		})

		it("removes existing copies of the destination folder's `node_modules`", func() {
			leftpad := filepath.Join(f.Build.Cache.Layer(detect.NPMDependency).Root, "node_modules", "left_pad")
			err = os.MkdirAll(leftpad, 0777)
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(filepath.Join(leftpad, "index.js"), []byte("leftpad"), 0666)
			Expect(err).To(BeNil())

			mockRunner.EXPECT().Run(gomock.Any(), gomock.Any()).Times(1)
			err = Npm.RebuildLayer(appRoot, cacheLayer)
			Expect(err).To(BeNil())
			Expect(filepath.Join(leftpad, "index.js")).ToNot(BeAnExistingFile())
		})

		it("copies the src directories node_modules folder", func() {
			leftpad := filepath.Join(appRoot, "node_modules", "left_pad")
			err = os.MkdirAll(leftpad, 0777)
			Expect(err).To(BeNil())

			mockRunner.EXPECT().Run(gomock.Any(), gomock.Any()).Times(1)

			err = Npm.RebuildLayer(appRoot, cacheLayer)

			Expect(err).To(BeNil())
			Expect(filepath.Join(cacheLayer, "node_modules", "left_pad")).To(BeAnExistingFile())
		})

		it("rebuilds in dst", func() {
			mockRunner.EXPECT().Run(cacheLayer, "rebuild").Times(1)

			err = Npm.RebuildLayer(appRoot, cacheLayer)
			Expect(err).To(BeNil())
		})
	})

	when("InstallToLayer", func() {
		it.Before(func() {
			err = os.MkdirAll(appRoot, 0777)
			Expect(err).To(BeNil())

			err = os.MkdirAll(filepath.Join(appRoot, "node_modules"), 0777)
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(filepath.Join(appRoot, "package.json"), []byte("package json"), 0666)
			Expect(err).To(BeNil())
		})

		it("run NPM install in the app dir", func() {
			installCommand := []string{"install", "--unsafe-perm", "--cache", fmt.Sprintf("%s/npm-cache", appRoot)}
			mockRunner.EXPECT().Run(appRoot, installCommand).Times(1)

			err = Npm.InstallToLayer(appRoot, cacheLayer)
			Expect(err).To(BeNil())
		})
	})

	when("CleanAndCopyToDst", func() {
		var original_bytes []byte
		var dest_file string
		it.Before(func() {
			cacheLayer = f.Build.Cache.Layer(detect.NPMDependency).Root
			rightpad := filepath.Join(cacheLayer, "node_modules", "rightpad")
			original_bytes = []byte("package json")

			err = os.MkdirAll(filepath.Join(f.Build.Launch.Root, "node_modules"), 0777)
			dest_file = filepath.Join(f.Build.Launch.Root, "node_modules", "old_modules")

			err = os.MkdirAll(filepath.Join(cacheLayer, "node_modules"), 0777)
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(rightpad, original_bytes, 0666)
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(dest_file, original_bytes, 0666)
			Expect(err).To(BeNil())

		})

		it("copies modules from src to dst", func() {
			copy_dest := filepath.Join(f.Build.Launch.Root, "node_modules", "rightpad")

			err = Npm.CleanAndCopyToDst(cacheLayer, f.Build.Launch.Root)
			Expect(err).To(BeNil())

			copied_bytes, err := ioutil.ReadFile(copy_dest)
			Expect(err).To(BeNil())
			Expect(copied_bytes).To(Equal(original_bytes))
		})
		it("removes old content in dst before copying", func() {
			Expect(dest_file).To(BeAnExistingFile())
			err = Npm.CleanAndCopyToDst(cacheLayer, f.Build.Launch.Root)
			Expect(err).To(BeNil())
			Expect(dest_file).ToNot(BeAnExistingFile())
		})

	})
}
