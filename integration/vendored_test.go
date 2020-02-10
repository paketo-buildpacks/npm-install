package integration_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/occam"
	"github.com/sclevine/spec"

	. "github.com/cloudfoundry/occam/matchers"
	. "github.com/onsi/gomega"
)

func testVendored(t *testing.T, context spec.G, it spec.S) {
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

	context("when the node_modules are vendored", func() {
		var (
			image     occam.Image
			container occam.Container

			name string
		)

		it.Before(func() {
			var err error
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
		})

		it("builds a working OCI image for a simple app", func() {
			var err error
			image, _, err = pack.Build.
				WithNoPull().
				WithBuildpacks(nodeURI, npmURI).
				Execute(name, filepath.Join("testdata", "vendored"))
			Expect(err).NotTo(HaveOccurred())

			container, err = docker.Container.Run.Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			Eventually(container, "5s").Should(BeAvailable())

			response, err := http.Get(fmt.Sprintf("http://localhost:%s", container.HostPort()))
			Expect(err).NotTo(HaveOccurred())
			Expect(response.StatusCode).To(Equal(http.StatusOK))

			content, err := ioutil.ReadAll(response.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("Hello, World!"))
		})

		context("when the npm and node buildpacks are cached", func() {
			it("does not reach out to the internet", func() {
				var err error
				image, _, err = pack.Build.
					WithNoPull().
					WithBuildpacks(nodeCachedURI, npmCachedURI).
					WithNetwork("none").
					Execute(name, filepath.Join("testdata", "vendored"))
				Expect(err).NotTo(HaveOccurred())

				container, err = docker.Container.Run.Execute(image.ID)
				Expect(err).NotTo(HaveOccurred())

				Eventually(container).Should(BeAvailable())

				response, err := http.Get(fmt.Sprintf("http://localhost:%s", container.HostPort()))
				Expect(err).NotTo(HaveOccurred())
				Expect(response.StatusCode).To(Equal(http.StatusOK))
			})
		})
	})
}
