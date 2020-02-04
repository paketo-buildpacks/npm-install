package integration_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/occam"
	"github.com/sclevine/spec"

	. "github.com/cloudfoundry/occam/matchers"
	. "github.com/onsi/gomega"
)

func testLogging(t *testing.T, context spec.G, it spec.S) {
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

	context("when the buildpack is run with pack build", func() {
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

		it.Focus("logs useful information for the user", func() {
			var err error
			var logs fmt.Stringer
			image, logs, err = pack.WithNoColor().Build.
				WithBuildpacks(nodeURI, npmURI).
				Execute(name, filepath.Join("testdata", "simple_app"))
			Expect(err).NotTo(HaveOccurred(), logs.String)

			container, err = docker.Container.Run.Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			Eventually(container).Should(BeAvailable())

			buildpackVersion, err := GetGitVersion()
			Expect(err).ToNot(HaveOccurred())

			sequence := []interface{}{
				//Title and version
				fmt.Sprintf("NPM Buildpack %s", buildpackVersion),
				//Resolve build process based on artifacts present"
				"  Resolving installation process",
				"    Process Inputs",
				"      package-lock.json -->",
				"      node_modules -->",
				"",
				//print selection based on artifacts
				MatchRegexp(`    Selected NPM build process:`),

				//execute chosen build process
				"  Executing build process",
				//pre install scripts, if present

				//post install scripts, if present
			}

			splitLogs := GetBuildLogs(logs.String())
			Expect(splitLogs).To(ContainSequence(sequence))
		})
	})
}
