package integration_test

import (
	"fmt"
	"os"
	"path/filepath"
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

	context("when a .npmrc file is in the application root directory", func() {
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

		it("is respected during npm install", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "npmrc"))
			Expect(err).NotTo(HaveOccurred())

			var logs fmt.Stringer
			image, logs, err = pack.Build.
				WithBuildpacks(nodeCachedURI, npmCachedURI).
				WithNoPull().
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String)

			container, err = docker.Container.Run.
				WithCommand("stat /workspace/postinstall.log").
				Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() string {
				logs, _ := docker.Container.Logs.Execute(container.ID)
				return logs.String()
			}).Should(ContainSubstring("No such file"))
		})
	})
}
