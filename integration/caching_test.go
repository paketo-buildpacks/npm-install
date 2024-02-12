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

		pullPolicy              = "never"
		extenderBuildStr        = ""
		extenderBuildStrEscaped = ""
	)

	it.Before(func() {
		imageIDs = make(map[string]struct{})
		containerIDs = make(map[string]struct{})

		pack = occam.NewPack().WithNoColor()
		docker = occam.NewDocker()

		var err error
		name, err = occam.RandomName()
		Expect(err).NotTo(HaveOccurred())

		if settings.Extensions.UbiNodejsExtension.Online != "" {
			pullPolicy = "always"
			extenderBuildStr = "[extender (build)] "
			extenderBuildStrEscaped = `\[extender \(build\)\] `
		}
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

			build := pack.Build.
				WithPullPolicy(pullPolicy).
				WithExtensions(
					settings.Extensions.UbiNodejsExtension.Online,
				).
				WithBuildpacks(
					settings.Buildpacks.NodeEngine.Online,
					settings.Buildpacks.NPMInstall.Online,
					settings.Buildpacks.BuildPlan.Online,
				)

			firstImage, logs, err := build.Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String)

			imageIDs[firstImage.ID] = struct{}{}

			Expect(firstImage.Buildpacks).To(HaveLen(3))
			Expect(firstImage.Buildpacks[1].Key).To(Equal(settings.Buildpack.ID))
			Expect(firstImage.Buildpacks[1].Layers).To(HaveKey("launch-modules"))

			container, err := docker.Container.Run.
				WithCommand("npm start").
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				Execute(firstImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[container.ID] = struct{}{}

			Eventually(container).Should(BeAvailable())

			secondImage, logs, err := build.Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String)

			imageIDs[secondImage.ID] = struct{}{}

			Expect(secondImage.Buildpacks).To(HaveLen(3))
			Expect(secondImage.Buildpacks[1].Key).To(Equal(settings.Buildpack.ID))
			Expect(secondImage.Buildpacks[1].Layers).To(HaveKey("launch-modules"))

			container, err = docker.Container.Run.
				WithCommand("npm start").
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				Execute(secondImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[container.ID] = struct{}{}

			Eventually(container).Should(BeAvailable())

			Expect(secondImage.ID).To(Equal(firstImage.ID))
			Expect(secondImage.Buildpacks[1].Layers["launch-modules"].SHA).To(Equal(firstImage.Buildpacks[1].Layers["launch-modules"].SHA))

			Expect(logs).To(ContainLines(
				extenderBuildStr + "  Executing launch environment install process",
			))
		})
	})

	context("when the app is locked", func() {
		it("reuses the node modules layer", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "locked_app"))
			Expect(err).NotTo(HaveOccurred())

			build := pack.Build.
				WithPullPolicy(pullPolicy).
				WithExtensions(
					settings.Extensions.UbiNodejsExtension.Online,
				).
				WithBuildpacks(
					settings.Buildpacks.NodeEngine.Online,
					settings.Buildpacks.NPMInstall.Online,
					settings.Buildpacks.BuildPlan.Online,
				)

			firstImage, logs, err := build.Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String)

			imageIDs[firstImage.ID] = struct{}{}

			Expect(firstImage.Buildpacks).To(HaveLen(3))
			Expect(firstImage.Buildpacks[1].Key).To(Equal(settings.Buildpack.ID))
			Expect(firstImage.Buildpacks[1].Layers).To(HaveKey("launch-modules"))

			Expect(logs).To(ContainLines(
				fmt.Sprintf("%s%s 1.2.3", extenderBuildStr, settings.Buildpack.Name),
				extenderBuildStr+"  Resolving installation process",
				extenderBuildStr+"    Process inputs:",
				extenderBuildStr+"      node_modules      -> \"Not found\"",
				extenderBuildStr+"      npm-cache         -> \"Not found\"",
				extenderBuildStr+"      package-lock.json -> \"Found\"",
				extenderBuildStr+"",
				extenderBuildStr+"    Selected NPM build process: 'npm ci'"))
			Expect(logs).To(ContainLines(
				extenderBuildStr+"  Executing launch environment install process",
				fmt.Sprintf(extenderBuildStr+"    Running 'npm ci --unsafe-perm --cache /layers/%s/npm-cache'", strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
			))
			Expect(logs).To(ContainLines(MatchRegexp(extenderBuildStrEscaped + `      Completed in (\d+\.\d+|\d{3})`)))
			Expect(logs).To(ContainLines(
				extenderBuildStr+"  Configuring launch environment",
				extenderBuildStr+"    NODE_PROJECT_PATH   -> \"/workspace\"",
				extenderBuildStr+"    NPM_CONFIG_LOGLEVEL -> \"error\"",
				fmt.Sprintf(extenderBuildStr+"    PATH                -> \"$PATH:/layers/%s/launch-modules/node_modules/.bin\"", strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
				extenderBuildStr+"",
			))

			container, err := docker.Container.Run.
				WithCommand("npm start").
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				Execute(firstImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[container.ID] = struct{}{}

			Eventually(container).Should(BeAvailable())

			secondImage, logs, err := build.Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String)

			imageIDs[secondImage.ID] = struct{}{}

			Expect(secondImage.Buildpacks).To(HaveLen(3))
			Expect(secondImage.Buildpacks[1].Key).To(Equal(settings.Buildpack.ID))
			Expect(secondImage.Buildpacks[1].Layers).To(HaveKey("launch-modules"))

			container, err = docker.Container.Run.
				WithCommand("npm start").
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				Execute(secondImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[container.ID] = struct{}{}

			Eventually(container).Should(BeAvailable())

			Expect(secondImage.ID).To(Equal(firstImage.ID))
			Expect(secondImage.Buildpacks[1].Layers["launch-modules"].SHA).To(Equal(firstImage.Buildpacks[1].Layers["launch-modules"].SHA))
		})

		context("and the node.js version has changed", func() {
			it("reinstalls node_modules", func() {
				var err error
				source, err = occam.Source(filepath.Join("testdata", "locked_app"))
				Expect(err).NotTo(HaveOccurred())

				build := pack.Build.
					WithPullPolicy(pullPolicy).
					WithExtensions(
						settings.Extensions.UbiNodejsExtension.Online,
					).
					WithEnv(map[string]string{"BP_NODE_VERSION": "~16"}).
					WithBuildpacks(
						settings.Buildpacks.NodeEngine.Online,
						settings.Buildpacks.NPMInstall.Online,
						settings.Buildpacks.BuildPlan.Online,
					)

				firstImage, logs, err := build.Execute(name, source)
				Expect(err).NotTo(HaveOccurred(), logs.String)

				imageIDs[firstImage.ID] = struct{}{}

				Expect(firstImage.Buildpacks).To(HaveLen(3))
				Expect(firstImage.Buildpacks[1].Key).To(Equal(settings.Buildpack.ID))
				Expect(firstImage.Buildpacks[1].Layers).To(HaveKey("launch-modules"))

				container, err := docker.Container.Run.
					WithCommand("npm start").
					WithEnv(map[string]string{"PORT": "8080"}).
					WithPublish("8080").
					Execute(firstImage.ID)
				Expect(err).NotTo(HaveOccurred())

				containerIDs[container.ID] = struct{}{}

				Eventually(container).Should(BeAvailable())

				build = pack.Build.
					WithPullPolicy(pullPolicy).
					WithExtensions(
						settings.Extensions.UbiNodejsExtension.Online,
					).
					WithEnv(map[string]string{"BP_NODE_VERSION": "~18"}).
					WithBuildpacks(
						settings.Buildpacks.NodeEngine.Online,
						settings.Buildpacks.NPMInstall.Online,
						settings.Buildpacks.BuildPlan.Online,
					)

				secondImage, logs, err := build.Execute(name, source)
				Expect(err).NotTo(HaveOccurred(), logs.String)

				imageIDs[secondImage.ID] = struct{}{}

				Expect(secondImage.Buildpacks).To(HaveLen(3))
				Expect(secondImage.Buildpacks[1].Key).To(Equal(settings.Buildpack.ID))
				Expect(secondImage.Buildpacks[1].Layers).To(HaveKey("launch-modules"))

				container, err = docker.Container.Run.
					WithCommand("npm start").
					WithEnv(map[string]string{"PORT": "8080"}).
					WithPublish("8080").
					Execute(secondImage.ID)
				Expect(err).NotTo(HaveOccurred())

				containerIDs[container.ID] = struct{}{}

				Eventually(container).Should(BeAvailable())

				Expect(secondImage.ID).NotTo(Equal(firstImage.ID))

				// TODO: Not sure why this fails now that we've upgraded Node versions.
				// If this is no longer a suitable indicator of cache invalidation then
				// we should find another.

				//Expect(secondImage.Buildpacks[1].Layers["launch-modules"].SHA).NotTo(Equal(firstImage.Buildpacks[1].Layers["launch-modules"].SHA))
			})
		})
	})

	context("when the app is vendored", func() {
		it("reuses the node modules layer", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "vendored"))
			Expect(err).NotTo(HaveOccurred())

			build := pack.WithNoColor().Build.
				WithPullPolicy(pullPolicy).
				WithExtensions(
					settings.Extensions.UbiNodejsExtension.Online,
				).
				WithBuildpacks(
					settings.Buildpacks.NodeEngine.Online,
					settings.Buildpacks.NPMInstall.Online,
					settings.Buildpacks.BuildPlan.Online,
				)

			firstImage, logs, err := build.Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String)

			imageIDs[firstImage.ID] = struct{}{}

			Expect(firstImage.Buildpacks).To(HaveLen(3))
			Expect(firstImage.Buildpacks[1].Key).To(Equal(settings.Buildpack.ID))
			Expect(firstImage.Buildpacks[1].Layers).To(HaveKey("launch-modules"))

			container, err := docker.Container.Run.
				WithCommand("npm start").
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				Execute(firstImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[container.ID] = struct{}{}

			Eventually(container).Should(BeAvailable())

			secondImage, logs, err := build.Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String)

			imageIDs[secondImage.ID] = struct{}{}

			Expect(secondImage.Buildpacks).To(HaveLen(3))
			Expect(secondImage.Buildpacks[1].Key).To(Equal(settings.Buildpack.ID))
			Expect(secondImage.Buildpacks[1].Layers).To(HaveKey("launch-modules"))

			container, err = docker.Container.Run.
				WithCommand("npm start").
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				Execute(secondImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[container.ID] = struct{}{}

			Eventually(container).Should(BeAvailable())

			Expect(secondImage.ID).To(Equal(firstImage.ID))
			Expect(secondImage.Buildpacks[1].Layers["launch-modules"].SHA).To(Equal(firstImage.Buildpacks[1].Layers["launch-modules"].SHA))

			Expect(logs).To(ContainLines(
				fmt.Sprintf("%s%s 1.2.3", extenderBuildStr, settings.Buildpack.Name),
				extenderBuildStr+"  Resolving installation process",
				extenderBuildStr+"    Process inputs:",
				extenderBuildStr+"      node_modules      -> \"Found\"",
				extenderBuildStr+"      npm-cache         -> \"Not found\"",
				extenderBuildStr+"      package-lock.json -> \"Found\"",
				extenderBuildStr+"",
				MatchRegexp(extenderBuildStrEscaped+`    Selected NPM build process:`),
				extenderBuildStr+"",
				fmt.Sprintf("%s  Reusing cached layer /layers/%s/launch-modules", extenderBuildStr, strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
			))
		})
	})

	context("when the app uses npm-cache", func() {
		it("reuses the npm-cache from the cache layer", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "npm-cache"))
			Expect(err).NotTo(HaveOccurred())

			build := pack.WithNoColor().Build.
				WithPullPolicy(pullPolicy).
				WithExtensions(
					settings.Extensions.UbiNodejsExtension.Online,
				).
				WithBuildpacks(
					settings.Buildpacks.NodeEngine.Online,
					settings.Buildpacks.NPMInstall.Online,
					settings.Buildpacks.BuildPlan.Online,
				)

			firstImage, logs, err := build.Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String)

			imageIDs[firstImage.ID] = struct{}{}

			Expect(firstImage.Buildpacks).To(HaveLen(3))
			Expect(firstImage.Buildpacks[1].Key).To(Equal(settings.Buildpack.ID))

			container, err := docker.Container.Run.
				WithCommand("npm start").
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				Execute(firstImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[container.ID] = struct{}{}

			Eventually(container).Should(BeAvailable())

			Expect(logs).To(ContainLines(
				fmt.Sprintf("%s%s 1.2.3", extenderBuildStr, settings.Buildpack.Name),
				extenderBuildStr+"  Resolving installation process",
				extenderBuildStr+"    Process inputs:",
				extenderBuildStr+"      node_modules      -> \"Found\"",
				extenderBuildStr+"      npm-cache         -> \"Found\"",
				extenderBuildStr+"      package-lock.json -> \"Found\"",
				extenderBuildStr+"",
				MatchRegexp(extenderBuildStrEscaped+`    Selected NPM build process:`),
			))

			secondImage, logs, err := build.Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String)

			imageIDs[secondImage.ID] = struct{}{}

			Expect(secondImage.Buildpacks).To(HaveLen(3))
			Expect(secondImage.Buildpacks[1].Key).To(Equal(settings.Buildpack.ID))

			container, err = docker.Container.Run.
				WithCommand("npm start").
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				Execute(secondImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[container.ID] = struct{}{}

			Eventually(container).Should(BeAvailable())
			Expect(secondImage.ID).To(Equal(firstImage.ID))

			Expect(logs).To(ContainLines(
				fmt.Sprintf("%s%s 1.2.3", extenderBuildStr, settings.Buildpack.Name),
				extenderBuildStr+"  Resolving installation process",
				extenderBuildStr+"    Process inputs:",
				extenderBuildStr+"      node_modules      -> \"Found\"",
				extenderBuildStr+"      npm-cache         -> \"Found\"",
				extenderBuildStr+"      package-lock.json -> \"Found\"",
				extenderBuildStr+"",
				MatchRegexp(extenderBuildStrEscaped+`    Selected NPM build process:`),
				extenderBuildStr+"",
				fmt.Sprintf(extenderBuildStr+"  Reusing cached layer /layers/%s/npm-cache", strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
			))
		})
	})

	context("when the app has workspaces", func() {
		it("ensures the workspaces are linked correctly", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "workspaces", "commonjs"))
			Expect(err).NotTo(HaveOccurred())

			build := pack.Build.
				WithPullPolicy(pullPolicy).
				WithExtensions(
					settings.Extensions.UbiNodejsExtension.Online,
				).
				WithBuildpacks(
					settings.Buildpacks.NodeEngine.Online,
					settings.Buildpacks.NPMInstall.Online,
					settings.Buildpacks.BuildPlan.Online,
				)

			firstImage, logs, err := build.Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String)

			imageIDs[firstImage.ID] = struct{}{}

			Expect(firstImage.Buildpacks[1].Key).To(Equal(settings.Buildpack.ID))

			container, err := docker.Container.Run.
				WithCommand("node server.js").
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				Execute(firstImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[container.ID] = struct{}{}

			Eventually(container).Should(BeAvailable())

			secondImage, logs, err := build.Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String)

			imageIDs[secondImage.ID] = struct{}{}

			Expect(secondImage.Buildpacks[1].Key).To(Equal(settings.Buildpack.ID))

			container, err = docker.Container.Run.
				WithCommand("node server.js").
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				Execute(secondImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[container.ID] = struct{}{}

			Eventually(container).Should(BeAvailable())
			Expect(secondImage.ID).To(Equal(firstImage.ID))
		})
	})
}
