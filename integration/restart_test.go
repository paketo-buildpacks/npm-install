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

func testRestart(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

		pack   occam.Pack
		docker occam.Docker

		image     occam.Image
		container occam.Container

		name    string
		source  string
		sbomDir string

		pullPolicy = "never"
	)

	it.Before(func() {
		pack = occam.NewPack()
		docker = occam.NewDocker()

		var err error
		name, err = occam.RandomName()
		Expect(err).NotTo(HaveOccurred())

		sbomDir, err = os.MkdirTemp("", "sbom")
		Expect(err).NotTo(HaveOccurred())
		Expect(os.Chmod(sbomDir, os.ModePerm)).To(Succeed())

		if settings.Extensions.UbiNodejsExtension.Online != "" {
			pullPolicy = "always"
		}
	})

	it.After(func() {
		Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
		Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
		Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
		Expect(os.RemoveAll(source)).To(Succeed())
		Expect(os.RemoveAll(sbomDir)).To(Succeed())
	})

	it("allows the process to be restarted", func() {
		var err error
		source, err = occam.Source(filepath.Join("testdata", "simple_app"))
		Expect(err).NotTo(HaveOccurred())

		image, _, err = pack.Build.
			WithExtensions(
				settings.Extensions.UbiNodejsExtension.Online,
			).
			WithBuildpacks(
				settings.Buildpacks.NodeEngine.Online,
				settings.Buildpacks.NPMInstall.Online,
				settings.Buildpacks.BuildPlan.Online,
			).
			WithPullPolicy(pullPolicy).
			WithSBOMOutputDir(sbomDir).
			Execute(name, source)
		Expect(err).NotTo(HaveOccurred())

		container, err = docker.Container.Run.
			WithDirect().
			WithCommand("node").
			WithCommandArgs([]string{"server.js"}).
			WithEnv(map[string]string{"PORT": "8080"}).
			WithPublish("8080:8080").
			Execute(image.ID)
		Expect(err).NotTo(HaveOccurred())

		Eventually(container).Should(Serve("Hello World!"), func() string {
			logs, _ := docker.Container.Logs.Execute(container.ID)
			return logs.String()
		})

		err = docker.Container.Restart.Execute(container.ID)
		Expect(err).NotTo(HaveOccurred())

		Eventually(container).Should(Serve("Hello World!"))
	})
}
