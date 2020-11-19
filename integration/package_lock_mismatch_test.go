package integration_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testPackageLockMismatch(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

		pack         occam.Pack
		docker       occam.Docker
		imageIDs     map[string]struct{}
		containerIDs map[string]struct{}

		name   string
		source string
	)

	it.Before(func() {
		imageIDs = make(map[string]struct{})
		containerIDs = make(map[string]struct{})

		pack = occam.NewPack().WithNoColor()
		docker = occam.NewDocker()

		var err error
		name, err = occam.RandomName()
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		for id := range containerIDs {
			Expect(docker.Container.Remove.Execute(id)).To(Succeed())
		}

		for id := range imageIDs {
			Expect(docker.Image.Remove.Execute(id)).To(Succeed())
		}

		Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
		Expect(os.RemoveAll(source)).To(Succeed())
	})

	context("when package.json and package-lock.json are mismatched on first build", func() {
		it("build must fail", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "locked_app"))
			Expect(err).NotTo(HaveOccurred())

			// manipulate package.json
			contents, err := ioutil.ReadFile(filepath.Join(source, "package.json"))
			Expect(err).NotTo(HaveOccurred())

			err = ioutil.WriteFile(filepath.Join(source, "package.json"),
				[]byte(strings.ReplaceAll(string(contents),
					`"dependencies": {`,
					`"dependencies": { "logfmt": "~1.1.2",`,
				)), 0644)
			Expect(err).NotTo(HaveOccurred())

			build := pack.Build.WithPullPolicy("never").WithBuildpacks(nodeURI, buildpackURI, buildPlanURI)
			_, logs, err := build.Execute(name, source)
			Expect(err).To(HaveOccurred(), logs.String)

			Expect(logs).To(ContainSubstring(
				"Please update your lock file with `npm install` before continuing",
			))
		})
	})

	context("when package.json and package-lock.json are mismatched on second build", func() {
		it("second build must fail and not reuse the layer", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "locked_app"))
			Expect(err).NotTo(HaveOccurred())

			build := pack.Build.WithPullPolicy("never").WithBuildpacks(nodeURI, buildpackURI, buildPlanURI)

			firstImage, logs, err := build.Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String)

			container, err := docker.Container.Run.WithCommand("npm start").Execute(firstImage.ID)
			Expect(err).NotTo(HaveOccurred())
			Eventually(container).Should(BeAvailable())

			// manipulate package.json
			contents, err := ioutil.ReadFile(filepath.Join(source, "package.json"))
			Expect(err).NotTo(HaveOccurred())

			err = ioutil.WriteFile(filepath.Join(source, "package.json"),
				[]byte(strings.ReplaceAll(string(contents),
					`"dependencies": {`,
					`"dependencies": { "logfmt": "~1.1.2",`,
				)), 0644)
			Expect(err).NotTo(HaveOccurred())

			_, logs, err = build.Execute(name, source)
			Expect(err).To(HaveOccurred(), logs.String)

			Expect(logs).To(ContainSubstring(
				"Please update your lock file with `npm install` before continuing",
			))
		})
	})
}
