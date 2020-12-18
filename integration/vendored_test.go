package integration_test

import (
	"fmt"
	"io/ioutil"
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

func testVendored(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

		pack   occam.Pack
		docker occam.Docker
	)

	it.Before(func() {
		pack = occam.NewPack().WithNoColor()
		docker = occam.NewDocker()
	})

	context("when the node_modules are vendored", func() {
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

			source, err = occam.Source(filepath.Join("testdata", "vendored"))
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		it("builds a working OCI image for a simple app", func() {
			var (
				err  error
				logs fmt.Stringer
			)

			image, logs, err = pack.Build.
				WithPullPolicy("never").
				WithBuildpacks(
					nodeURI,
					buildpackURI,
					buildPlanURI,
				).
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred())

			container, err = docker.Container.Run.
				WithCommand("npm start").
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			Eventually(container).Should(BeAvailable())

			response, err := http.Get(fmt.Sprintf("http://localhost:%s", container.HostPort("8080")))
			Expect(err).NotTo(HaveOccurred())
			Expect(response.StatusCode).To(Equal(http.StatusOK))

			content, err := ioutil.ReadAll(response.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("Hello, World!"))

			Expect(logs).To(ContainLines(
				fmt.Sprintf("%s 1.2.3", buildpackInfo.Buildpack.Name),
				"  Resolving installation process",
				"    Process inputs:",
				"      node_modules      -> \"Found\"",
				"      npm-cache         -> \"Not found\"",
				"      package-lock.json -> \"Found\"",
				"",
				"    Selected NPM build process: 'npm rebuild'",
				"",
				"  Executing build process",
				"    Running 'npm run-script preinstall --if-present'",
				MatchRegexp(`    Running 'npm rebuild --nodedir=/layers/.+/node'`),
				"    Running 'npm run-script postinstall --if-present'",
				MatchRegexp(`      Completed in (\d+\.\d+|\d{3})`),
				"",
				"  Configuring environment",
				"    NPM_CONFIG_LOGLEVEL   -> \"error\"",
				"    NPM_CONFIG_PRODUCTION -> \"true\"",
				fmt.Sprintf("    PATH                  -> \"$PATH:/layers/%s/modules/node_modules/.bin\"", strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_")),
				"",
			))
		})

		context("when the npm and node buildpacks are cached", func() {
			it("does not reach out to the internet", func() {
				var err error
				image, _, err = pack.Build.
					WithPullPolicy("never").
					WithBuildpacks(
						nodeOfflineURI,
						buildpackOfflineURI,
						buildPlanURI,
					).
					WithNetwork("none").
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				container, err = docker.Container.Run.
					WithCommand("npm start").
					WithEnv(map[string]string{"PORT": "8080"}).
					WithPublish("8080").
					Execute(image.ID)
				Expect(err).NotTo(HaveOccurred())

				Eventually(container).Should(BeAvailable())

				response, err := http.Get(fmt.Sprintf("http://localhost:%s", container.HostPort("8080")))
				Expect(err).NotTo(HaveOccurred())
				Expect(response.StatusCode).To(Equal(http.StatusOK))
			})
		})
	})
}
