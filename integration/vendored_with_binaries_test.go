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

func testVendoredWithBinaries(t *testing.T, context spec.G, it spec.S) {
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

	context("when the node_modules are vendored and contains a binary in .bin", func() {
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

		it("builds a working OCI image for a vendored app with binary", func() {
			var err error
			var logs fmt.Stringer
			image, logs, err = pack.WithVerbose().Build.
				WithBuildpacks(nodeURI, npmURI).
				Execute(name, filepath.Join("testdata", "vendored"))
			Expect(err).NotTo(HaveOccurred(), logs.String)

			container, err = docker.Container.Run.WithCommand(`figlet "PACKETO" -f Train && npm start`).Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			Eventually(container, "5s").Should(BeAvailable())

			response, err := http.Get(fmt.Sprintf("http://localhost:%s", container.HostPort()))
			Expect(err).NotTo(HaveOccurred())
			Expect(response.StatusCode).To(Equal(http.StatusOK))

			content, err := ioutil.ReadAll(response.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("Hello, World!"))
		})
	})
}
