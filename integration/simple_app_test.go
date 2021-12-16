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

func testSimpleApp(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

		pack   occam.Pack
		docker occam.Docker
	)

	it.Before(func() {
		pack = occam.NewPack()
		docker = occam.NewDocker()
	})

	context("when the node_modules are not vendored", func() {
		var (
			image      occam.Image
			container1 occam.Container
			container2 occam.Container
			container3 occam.Container

			name   string
			source string
		)

		it.Before(func() {
			var err error
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(docker.Container.Remove.Execute(container1.ID)).To(Succeed())
			Expect(docker.Container.Remove.Execute(container2.ID)).To(Succeed())
			Expect(docker.Container.Remove.Execute(container3.ID)).To(Succeed())
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		it("builds a working OCI image for a simple app", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "simple_app"))
			Expect(err).NotTo(HaveOccurred())

			image, _, err = pack.Build.
				WithBuildpacks(nodeURI, buildpackURI, buildPlanURI).
				WithPullPolicy("never").
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred())

			container1, err = docker.Container.Run.
				WithCommand(fmt.Sprintf("ls -alR /layers/%s/modules/node_modules && env", strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_"))).
				Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			cLogs := func() string {
				cLogs, err := docker.Container.Logs.Execute(container1.ID)
				Expect(err).NotTo(HaveOccurred())
				return cLogs.String()
			}
			Eventually(cLogs).Should(ContainSubstring("leftpad"))
			Eventually(cLogs).Should(ContainSubstring("NPM_CONFIG_LOGLEVEL=error"))

			container2, err = docker.Container.Run.
				WithCommand(fmt.Sprintf("cat /layers/sbom/launch/%s/modules/sbom.cdx.json", strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_"))).
				Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			cLogs = func() string {
				cLogs, err := docker.Container.Logs.Execute(container2.ID)
				Expect(err).NotTo(HaveOccurred())
				return cLogs.String()
			}
			Eventually(cLogs).Should(ContainSubstring(`"bomFormat": "CycloneDX"`))
			Eventually(cLogs).Should(ContainSubstring(`"name": "leftpad"`))

			// check that all expected SBOM files are present
			container3, err = docker.Container.Run.
				WithCommand(fmt.Sprintf("ls -al /layers/sbom/launch/%s/modules/",
					strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_"))).
				Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() string {
				cLogs, err := docker.Container.Logs.Execute(container3.ID)
				Expect(err).NotTo(HaveOccurred())
				return cLogs.String()
			}).Should(And(
				ContainSubstring("sbom.cdx.json"),
				ContainSubstring("sbom.spdx.json"),
				ContainSubstring("sbom.syft.json"),
			))
		})
	})
}
