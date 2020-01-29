package integration_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudfoundry/occam"
	"github.com/sclevine/spec"

	. "github.com/cloudfoundry/occam/matchers"
	. "github.com/onsi/gomega"
)

func testVersioning(t *testing.T, context spec.G, it spec.S) {
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

	context("when using a nvmrc file", func() {
		const nvmrcVersion = `12.\d+\.\d+`

		var (
			image     occam.Image
			container occam.Container

			name string
		)

		it.Before(func() {
			var err error
			name, err = occam.RandomName()
			Expect(err).ToNot(HaveOccurred())
		})

		it.After(func() {
			Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name)))
		})

		it("package.json takes precedence over it", func() {
			var err error
			image, _, err = pack.Build.
				WithBuildpacks(nodeURI, npmURI).
				Execute(name, filepath.Join("testdata", "with_nvmrc"))
			Expect(err).ToNot(HaveOccurred())

			container, err = docker.Container.Run.Execute(image.ID)
			Expect(err).ToNot(HaveOccurred())

			Eventually(container).Should(BeAvailable())

			response, err := http.Get(fmt.Sprintf("http://localhost:%s", container.HostPort()))
			Expect(err).NotTo(HaveOccurred())
			Expect(response.StatusCode).To(Equal(http.StatusOK))

			content, err := ioutil.ReadAll(response.Body)
			Expect(err).NotTo(HaveOccurred())

			body := strings.TrimSpace(string(content))
			Expect(body).To(MatchRegexp(`Hello, World! From node version: v\d+\.\d+.\d+`))
			Expect(body).NotTo(MatchRegexp(`Hello, World! From node version: v` + nvmrcVersion))
		})

		it("is honored if the package.json doesn't have an engine version", func() {
			var err error
			image, _, err = pack.Build.
				WithBuildpacks(nodeURI, npmURI).
				Execute(name, filepath.Join("testdata", "with_nvmrc_and_no_engine"))
			Expect(err).ToNot(HaveOccurred())

			container, err = docker.Container.Run.Execute(image.ID)
			Expect(err).ToNot(HaveOccurred())

			Eventually(container).Should(BeAvailable())

			response, err := http.Get(fmt.Sprintf("http://localhost:%s", container.HostPort()))
			Expect(err).NotTo(HaveOccurred())
			Expect(response.StatusCode).To(Equal(http.StatusOK))

			content, err := ioutil.ReadAll(response.Body)
			Expect(err).NotTo(HaveOccurred())

			body := strings.TrimSpace(string(content))
			Expect(body).To(MatchRegexp(`Hello, World! From node version: v` + nvmrcVersion))
		})
	})
}
