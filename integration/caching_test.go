package integration_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/occam"
	"github.com/sclevine/spec"

	. "github.com/cloudfoundry/occam/matchers"
	. "github.com/onsi/gomega"
)

func testCaching(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

		pack         occam.Pack
		docker       occam.Docker
		imageIDs     map[string]struct{}
		containerIDs map[string]struct{}

		imageName string
	)

	it.Before(func() {
		imageIDs = make(map[string]struct{})
		containerIDs = make(map[string]struct{})

		pack = occam.NewPack()
		docker = occam.NewDocker()

		var err error
		imageName, err = occam.RandomName()
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		for id, _ := range containerIDs {
			Expect(docker.Container.Remove.Execute(id)).To(Succeed())
		}

		for id, _ := range imageIDs {
			Expect(docker.Image.Remove.Execute(id)).To(Succeed())
		}

		Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(imageName))).To(Succeed())
	})

	context("when the app is not locked or vendored", func() {
		it("reinstalls node_modules", func() {
			sourcePath := filepath.Join("testdata", "simple_app")

			build := pack.Build.WithBuildpacks(nodeURI, npmURI)

			firstImage, logs, err := build.Execute(imageName, sourcePath)
			Expect(err).NotTo(HaveOccurred(), logs.String)

			imageIDs[firstImage.ID] = struct{}{}

			Expect(firstImage.Buildpacks).To(HaveLen(2))
			Expect(firstImage.Buildpacks[1].Key).To(Equal("org.cloudfoundry.npm"))
			Expect(firstImage.Buildpacks[1].Layers).To(HaveKey("modules"))

			container, err := docker.Container.Run.Execute(firstImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[container.ID] = struct{}{}

			Eventually(container, "5s").Should(BeAvailable())

			secondImage, logs, err := build.Execute(imageName, sourcePath)
			Expect(err).NotTo(HaveOccurred(), logs.String)

			imageIDs[secondImage.ID] = struct{}{}

			Expect(secondImage.Buildpacks).To(HaveLen(2))
			Expect(secondImage.Buildpacks[1].Key).To(Equal("org.cloudfoundry.npm"))
			Expect(secondImage.Buildpacks[1].Layers).To(HaveKey("modules"))

			container, err = docker.Container.Run.Execute(secondImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[container.ID] = struct{}{}

			Eventually(container, "5s").Should(BeAvailable())

			Expect(secondImage.ID).NotTo(Equal(firstImage.ID))
			Expect(secondImage.Buildpacks[1].Layers["modules"].Metadata["built_at"]).NotTo(Equal(firstImage.Buildpacks[1].Layers["modules"].Metadata["built_at"]))
		})
	})

	context("when the app is locked", func() {
		it("reuses the node modules layer", func() {
			sourcePath := filepath.Join("testdata", "locked_app")

			build := pack.Build.WithBuildpacks(nodeURI, npmURI)

			firstImage, logs, err := build.Execute(imageName, sourcePath)
			Expect(err).NotTo(HaveOccurred(), logs.String)

			imageIDs[firstImage.ID] = struct{}{}

			Expect(firstImage.Buildpacks).To(HaveLen(2))
			Expect(firstImage.Buildpacks[1].Key).To(Equal("org.cloudfoundry.npm"))
			Expect(firstImage.Buildpacks[1].Layers).To(HaveKey("modules"))

			container, err := docker.Container.Run.Execute(firstImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[container.ID] = struct{}{}

			Eventually(container, "5s").Should(BeAvailable())

			secondImage, logs, err := build.Execute(imageName, sourcePath)
			Expect(err).NotTo(HaveOccurred(), logs.String)

			imageIDs[secondImage.ID] = struct{}{}

			Expect(secondImage.Buildpacks).To(HaveLen(2))
			Expect(secondImage.Buildpacks[1].Key).To(Equal("org.cloudfoundry.npm"))
			Expect(secondImage.Buildpacks[1].Layers).To(HaveKey("modules"))

			container, err = docker.Container.Run.Execute(secondImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[container.ID] = struct{}{}

			Eventually(container, "5s").Should(BeAvailable())

			Expect(secondImage.ID).To(Equal(firstImage.ID))
			Expect(secondImage.Buildpacks[1].Layers["modules"].SHA).To(Equal(firstImage.Buildpacks[1].Layers["modules"].SHA))
			Expect(secondImage.Buildpacks[1].Layers["modules"].Metadata["built_at"]).To(Equal(firstImage.Buildpacks[1].Layers["modules"].Metadata["built_at"]))
		})
	})

	context("when the app is vendored", func() {
		it("reuses the node modules layer", func() {
			sourcePath := filepath.Join("testdata", "vendored")

			build := pack.WithNoColor().Build.WithBuildpacks(nodeURI, npmURI)

			firstImage, logs, err := build.Execute(imageName, sourcePath)
			Expect(err).NotTo(HaveOccurred(), logs.String)

			imageIDs[firstImage.ID] = struct{}{}

			Expect(firstImage.Buildpacks).To(HaveLen(2))
			Expect(firstImage.Buildpacks[1].Key).To(Equal("org.cloudfoundry.npm"))
			Expect(firstImage.Buildpacks[1].Layers).To(HaveKey("modules"))

			container, err := docker.Container.Run.Execute(firstImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[container.ID] = struct{}{}

			Eventually(container, "5s").Should(BeAvailable())

			secondImage, logs, err := build.Execute(imageName, sourcePath)
			Expect(err).NotTo(HaveOccurred(), logs.String)

			imageIDs[secondImage.ID] = struct{}{}

			Expect(secondImage.Buildpacks).To(HaveLen(2))
			Expect(secondImage.Buildpacks[1].Key).To(Equal("org.cloudfoundry.npm"))
			Expect(secondImage.Buildpacks[1].Layers).To(HaveKey("modules"))

			container, err = docker.Container.Run.Execute(secondImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[container.ID] = struct{}{}

			Eventually(container, "5s").Should(BeAvailable())

			Expect(secondImage.ID).To(Equal(firstImage.ID))
			Expect(secondImage.Buildpacks[1].Layers["modules"].SHA).To(Equal(firstImage.Buildpacks[1].Layers["modules"].SHA))
			Expect(secondImage.Buildpacks[1].Layers["modules"].Metadata["built_at"]).To(Equal(firstImage.Buildpacks[1].Layers["modules"].Metadata["built_at"]))

			buildpackVersion, err := GetGitVersion()
			Expect(err).ToNot(HaveOccurred())

			sequence := []interface{}{
				//Title and version
				fmt.Sprintf("NPM Buildpack %s", buildpackVersion),
				//Resolve build process based on artifacts present"
				"  Resolving installation process",
				"    Process inputs:",
				"      node_modules      -> Found",
				"      npm-cache         -> Not found",
				"      package-lock.json -> Found",
				//print selection based on artifacts
				MatchRegexp(`    Selected NPM build process:`),
				"",
				"  Reusing cached layer /layers/org.cloudfoundry.npm/modules",
			}

			splitLogs := GetBuildLogs(logs.String())
			Expect(splitLogs).To(ContainSequence(sequence))
		})
	})
}
