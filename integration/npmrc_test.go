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
)

func testNpmrc(t *testing.T, context spec.G, it spec.S) {
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

	context("when an .npmrc is used to configure installation", func() {
		var (
			image     occam.Image
			container occam.Container

			name   string
			source string
		)

		it.Before(func() {
			var err error
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed(), fmt.Sprintf("failed removing container %#v\n", container))
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		context("when a .npmrc file is in the application root directory", func() {
			it("is respected during npm install", func() {
				var err error
				source, err = occam.Source(filepath.Join("testdata", "npmrc"))
				Expect(err).NotTo(HaveOccurred())

				var logs fmt.Stringer
				image, logs, err = pack.Build.
					WithBuildpacks(
						settings.Buildpacks.NodeEngine.Online,
						settings.Buildpacks.NPMInstall.Online,
						settings.Buildpacks.BuildPlan.Online,
					).
					WithPullPolicy("never").
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred(), logs.String)

				container, err = docker.Container.Run.
					WithEntrypoint("stat").
					WithCommand("/workspace/postinstall.log").
					Execute(image.ID)
				Expect(err).NotTo(HaveOccurred())

				Eventually(func() string {
					logs, _ := docker.Container.Logs.Execute(container.ID)
					return logs.String()
				}).Should(ContainSubstring("No such file"))
			})
		})
		context("when an npmrc service binding is provided", func() {
			var (
				binding string
				err     error
			)
			it.Before(func() {
				binding, err = os.MkdirTemp("", "bindingdir")
				Expect(err).NotTo(HaveOccurred())
				Expect(os.Chmod(binding, os.ModePerm)).To(Succeed())

				Expect(os.WriteFile(filepath.Join(binding, "type"), []byte("npmrc"), os.ModePerm)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(binding, ".npmrc"), []byte(`include=dev`), os.ModePerm)).To(Succeed())
			})

			it.After(func() {
				os.RemoveAll(binding)
			})

			it("builds a working OCI image that includes the dev dependency", func() {
				var err error
				source, err = occam.Source(filepath.Join("testdata", "simple_app"))
				Expect(err).NotTo(HaveOccurred())

				image, _, err = pack.Build.
					WithBuildpacks(
						settings.Buildpacks.NodeEngine.Online,
						settings.Buildpacks.NPMInstall.Online,
						settings.Buildpacks.BuildPlan.Online,
					).
					WithPullPolicy("never").
					WithEnv(map[string]string{
						"SERVICE_BINDING_ROOT": "/bindings",
					}).
					WithVolumes(fmt.Sprintf("%s:/bindings/npmrc", binding)).
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				container, err = docker.Container.Run.
					WithCommand(fmt.Sprintf("ls -alR /layers/%s/launch-modules/node_modules", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"))).
					Execute(image.ID)
				Expect(err).NotTo(HaveOccurred())

				cLogs := func() string {
					cLogs, err := docker.Container.Logs.Execute(container.ID)
					Expect(err).NotTo(HaveOccurred())
					return cLogs.String()
				}
				Eventually(cLogs).Should(ContainSubstring("leftpad"))
				Eventually(cLogs).Should(ContainSubstring("spellchecker-cli"))
			})
		})
	})
}
