package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testWorkspaces(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

		pack   occam.Pack
		docker occam.Docker
	)

	it.Before(func() {
		pack = occam.NewPack().WithVerbose()
		docker = occam.NewDocker()
	})

	context("when a package.json references workspaces", func() {
		var (
			image     occam.Image
			container occam.Container

			name   string
			source string
		)

		context("when using CommonJS module loading", func() {
			it.Before(func() {
				var err error
				name, err = occam.RandomName()
				Expect(err).NotTo(HaveOccurred())

				source, err = occam.Source(filepath.Join("testdata", "workspaces", "commonjs"))
				Expect(err).NotTo(HaveOccurred())
			})

			it.After(func() {
				Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
				Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
				Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
				Expect(os.RemoveAll(source)).To(Succeed())
			})

			it("ensures those modules are reachable", func() {
				var err error
				image, _, err = pack.Build.
					WithBuildpacks(
						settings.Buildpacks.NodeEngine.Online,
						settings.Buildpacks.NPMInstall.Online,
						settings.Buildpacks.BuildPlan.Online,
					).
					WithPullPolicy("never").
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				container, err = docker.Container.Run.
					WithCommand("node server.js").
					WithPublish("8080").
					WithEnv(map[string]string{
						"PORT": "8080",
					}).
					Execute(image.ID)
				Expect(err).NotTo(HaveOccurred())

				Eventually(container).Should(Serve("Hello World!").OnPort(8080))
			})
		})

		context("when using ECMAScript module loading", func() {
			it.Before(func() {
				var err error
				name, err = occam.RandomName()
				Expect(err).NotTo(HaveOccurred())

				source, err = occam.Source(filepath.Join("testdata", "workspaces", "ecmascript"))
				Expect(err).NotTo(HaveOccurred())
			})

			it.After(func() {
				Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
				Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
				Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
				Expect(os.RemoveAll(source)).To(Succeed())
			})

			it("ensures those modules are reachable", func() {
				var err error
				image, _, err = pack.Build.
					WithBuildpacks(
						settings.Buildpacks.NodeEngine.Online,
						settings.Buildpacks.NPMInstall.Online,
						settings.Buildpacks.BuildPlan.Online,
					).
					WithPullPolicy("never").
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				container, err = docker.Container.Run.
					WithCommand("node server.js").
					WithPublish("8080").
					WithEnv(map[string]string{
						"PORT": "8080",
					}).
					Execute(image.ID)
				Expect(err).NotTo(HaveOccurred())

				Eventually(container).Should(Serve("Hello World!").OnPort(8080))
			})
		})

		context("when using webpack", func() {
			it.Before(func() {
				var err error
				name, err = occam.RandomName()
				Expect(err).NotTo(HaveOccurred())

				source, err = occam.Source(filepath.Join("testdata", "workspaces", "webpack"))
				Expect(err).NotTo(HaveOccurred())
			})

			it.After(func() {
				Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
				Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
				Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
				Expect(os.RemoveAll(source)).To(Succeed())
			})

			it("ensures those modules are reachable", func() {
				var err error
				image, _, err = pack.Build.
					WithPullPolicy("never").
					WithBuildpacks(
						settings.Buildpacks.NodeEngine.Online,
						settings.Buildpacks.NPMInstall.Online,
						settings.Buildpacks.NodeRunScript.Online,
						settings.Buildpacks.NGINX.Online,
					).
					WithEnv(map[string]string{
						"BP_NODE_RUN_SCRIPTS": "build",
						"BP_WEB_SERVER":       "nginx",
						"BP_WEB_SERVER_ROOT":  "build",
					}).
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				container, err = docker.Container.Run.
					WithPublish("8080").
					WithEnv(map[string]string{
						"PORT": "8080",
					}).
					Execute(image.ID)
				Expect(err).NotTo(HaveOccurred())

				Eventually(container).Should(Serve(ContainSubstring("Hello World!")).OnPort(8080))
			})
		})
	})
}
