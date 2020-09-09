package integration_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testCaching(t *testing.T, context spec.G, it spec.S) {
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

	context("when the app is not locked or vendored", func() {
		it("reinstalls node_modules", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "simple_app"))
			Expect(err).NotTo(HaveOccurred())

			build := pack.Build.WithNoPull().WithBuildpacks(nodeURI, buildpackURI, buildPlanURI)

			firstImage, logs, err := build.Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String)

			imageIDs[firstImage.ID] = struct{}{}

			Expect(firstImage.Buildpacks).To(HaveLen(3))
			Expect(firstImage.Buildpacks[1].Key).To(Equal(buildpackInfo.Buildpack.ID))
			Expect(firstImage.Buildpacks[1].Layers).To(HaveKey("modules"))

			container, err := docker.Container.Run.WithCommand("npm start").Execute(firstImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[container.ID] = struct{}{}

			Eventually(container).Should(BeAvailable())

			secondImage, logs, err := build.Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String)

			imageIDs[secondImage.ID] = struct{}{}

			Expect(secondImage.Buildpacks).To(HaveLen(3))
			Expect(secondImage.Buildpacks[1].Key).To(Equal(buildpackInfo.Buildpack.ID))
			Expect(secondImage.Buildpacks[1].Layers).To(HaveKey("modules"))

			container, err = docker.Container.Run.WithCommand("npm start").Execute(secondImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[container.ID] = struct{}{}

			Eventually(container).Should(BeAvailable())

			Expect(secondImage.ID).NotTo(Equal(firstImage.ID))
			Expect(secondImage.Buildpacks[1].Layers["modules"].Metadata["built_at"]).NotTo(Equal(firstImage.Buildpacks[1].Layers["modules"].Metadata["built_at"]))
		})
	})

	context("when the app is locked", func() {
		it("reuses the node modules layer", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "locked_app"))
			Expect(err).NotTo(HaveOccurred())

			build := pack.Build.WithNoPull().WithBuildpacks(nodeURI, buildpackURI, buildPlanURI)

			firstImage, logs, err := build.Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String)

			imageIDs[firstImage.ID] = struct{}{}

			Expect(firstImage.Buildpacks).To(HaveLen(3))
			Expect(firstImage.Buildpacks[1].Key).To(Equal(buildpackInfo.Buildpack.ID))
			Expect(firstImage.Buildpacks[1].Layers).To(HaveKey("modules"))

			Expect(logs).To(ContainLines(
				fmt.Sprintf("%s 1.2.3", buildpackInfo.Buildpack.Name),
				"  Resolving installation process",
				"    Process inputs:",
				"      node_modules      -> \"Not found\"",
				"      npm-cache         -> \"Not found\"",
				"      package-lock.json -> \"Found\"",
				"",
				"    Selected NPM build process: 'npm ci'",
				"",
				"  Executing build process",
				fmt.Sprintf("    Running 'npm ci --unsafe-perm --cache /layers/%s/npm-cache'", strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_")),
				MatchRegexp(`      Completed in (\d+\.\d+|\d{3})`),
				"",
				"  Configuring environment",
				"    NPM_CONFIG_LOGLEVEL   -> \"error\"",
				"    NPM_CONFIG_PRODUCTION -> \"true\"",
				fmt.Sprintf("    PATH                  -> \"$PATH:/layers/%s/modules/node_modules/.bin\"", strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_")),
			))

			container, err := docker.Container.Run.WithCommand("npm start").Execute(firstImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[container.ID] = struct{}{}

			Eventually(container).Should(BeAvailable())

			secondImage, logs, err := build.Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String)

			imageIDs[secondImage.ID] = struct{}{}

			Expect(secondImage.Buildpacks).To(HaveLen(3))
			Expect(secondImage.Buildpacks[1].Key).To(Equal(buildpackInfo.Buildpack.ID))
			Expect(secondImage.Buildpacks[1].Layers).To(HaveKey("modules"))

			container, err = docker.Container.Run.WithCommand("npm start").Execute(secondImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[container.ID] = struct{}{}

			Eventually(container).Should(BeAvailable())

			Expect(secondImage.ID).To(Equal(firstImage.ID))
			Expect(secondImage.Buildpacks[1].Layers["modules"].SHA).To(Equal(firstImage.Buildpacks[1].Layers["modules"].SHA))
			Expect(secondImage.Buildpacks[1].Layers["modules"].Metadata["built_at"]).To(Equal(firstImage.Buildpacks[1].Layers["modules"].Metadata["built_at"]))
		})
	})

	context("when the app is vendored", func() {
		it("reuses the node modules layer", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "vendored"))
			Expect(err).NotTo(HaveOccurred())

			build := pack.WithNoColor().Build.WithNoPull().WithBuildpacks(nodeURI, buildpackURI, buildPlanURI)

			firstImage, logs, err := build.Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String)

			imageIDs[firstImage.ID] = struct{}{}

			Expect(firstImage.Buildpacks).To(HaveLen(3))
			Expect(firstImage.Buildpacks[1].Key).To(Equal(buildpackInfo.Buildpack.ID))
			Expect(firstImage.Buildpacks[1].Layers).To(HaveKey("modules"))

			container, err := docker.Container.Run.WithCommand("npm start").Execute(firstImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[container.ID] = struct{}{}

			Eventually(container).Should(BeAvailable())

			secondImage, logs, err := build.Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String)

			imageIDs[secondImage.ID] = struct{}{}

			Expect(secondImage.Buildpacks).To(HaveLen(3))
			Expect(secondImage.Buildpacks[1].Key).To(Equal(buildpackInfo.Buildpack.ID))
			Expect(secondImage.Buildpacks[1].Layers).To(HaveKey("modules"))

			container, err = docker.Container.Run.WithCommand("npm start").Execute(secondImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[container.ID] = struct{}{}

			Eventually(container).Should(BeAvailable())

			Expect(secondImage.ID).To(Equal(firstImage.ID))
			Expect(secondImage.Buildpacks[1].Layers["modules"].SHA).To(Equal(firstImage.Buildpacks[1].Layers["modules"].SHA))
			Expect(secondImage.Buildpacks[1].Layers["modules"].Metadata["built_at"]).To(Equal(firstImage.Buildpacks[1].Layers["modules"].Metadata["built_at"]))

			Expect(logs).To(ContainLines(
				fmt.Sprintf("%s 1.2.3", buildpackInfo.Buildpack.Name),
				"  Resolving installation process",
				"    Process inputs:",
				"      node_modules      -> \"Found\"",
				"      npm-cache         -> \"Not found\"",
				"      package-lock.json -> \"Found\"",
				"",
				MatchRegexp(`    Selected NPM build process:`),
				"",
				fmt.Sprintf("  Reusing cached layer /layers/%s/modules", strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_")),
			))
		})
	})
}
