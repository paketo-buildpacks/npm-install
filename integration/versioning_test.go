package integration_test

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
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
		const nvmrcVersion = `18.\d+\.\d+`

		var (
			image      occam.Image
			container  occam.Container
			name       string
			source     string
			pullPolicy = "never"
		)

		it.Before(func() {
			var err error
			name, err = occam.RandomName()
			Expect(err).ToNot(HaveOccurred())

			if settings.Extensions.UbiNodejsExtension.Online != "" {
				pullPolicy = "always"
			}
		})

		it.After(func() {
			Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name)))
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		it("package.json takes precedence over it", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "with_nvmrc"))
			Expect(err).ToNot(HaveOccurred())

			image, _, err = pack.Build.
				WithPullPolicy(pullPolicy).
				WithExtensions(
					settings.Extensions.UbiNodejsExtension.Online,
				).
				WithBuildpacks(
					settings.Buildpacks.NodeEngine.Online,
					settings.Buildpacks.NPMInstall.Online,
					settings.Buildpacks.BuildPlan.Online,
				).
				Execute(name, source)
			Expect(err).ToNot(HaveOccurred())

			container, err = docker.Container.Run.
				WithCommand("npm start").
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				Execute(image.ID)
			Expect(err).ToNot(HaveOccurred())

			Eventually(container).Should(BeAvailable())

			response, err := http.Get(fmt.Sprintf("http://localhost:%s", container.HostPort("8080")))
			Expect(err).NotTo(HaveOccurred())
			Expect(response.StatusCode).To(Equal(http.StatusOK))

			content, err := io.ReadAll(response.Body)
			Expect(err).NotTo(HaveOccurred())

			body := strings.TrimSpace(string(content))
			Expect(body).To(MatchRegexp(`Hello, World! From node version: v\d+\.\d+.\d+`))
			Expect(body).NotTo(MatchRegexp(`Hello, World! From node version: v` + nvmrcVersion))
		})

		it("is honored if the package.json doesn't have an engine version", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "with_nvmrc_and_no_engine"))
			Expect(err).ToNot(HaveOccurred())

			image, _, err = pack.Build.
				WithPullPolicy(pullPolicy).
				WithExtensions(
					settings.Extensions.UbiNodejsExtension.Online,
				).
				WithBuildpacks(
					settings.Buildpacks.NodeEngine.Online,
					settings.Buildpacks.NPMInstall.Online,
					settings.Buildpacks.BuildPlan.Online,
				).
				Execute(name, source)
			Expect(err).ToNot(HaveOccurred())

			container, err = docker.Container.Run.
				WithCommand("npm start").
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				Execute(image.ID)
			Expect(err).ToNot(HaveOccurred())

			Eventually(container).Should(BeAvailable())

			response, err := http.Get(fmt.Sprintf("http://localhost:%s", container.HostPort("8080")))
			Expect(err).NotTo(HaveOccurred())
			Expect(response.StatusCode).To(Equal(http.StatusOK))

			content, err := io.ReadAll(response.Body)
			Expect(err).NotTo(HaveOccurred())

			body := strings.TrimSpace(string(content))
			Expect(body).To(MatchRegexp(`Hello, World! From node version: v` + nvmrcVersion))
		})
	})
}
