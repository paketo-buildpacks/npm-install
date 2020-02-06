package integration_test

import (
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testUnmetDependencies(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		pack   occam.Pack
		docker occam.Docker

		name string
	)

	it.Before(func() {
		pack = occam.NewPack()
		docker = occam.NewDocker()

		var err error
		name, err = occam.RandomName()
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
	})

	context("when the package manager is npm", func() {
		it("warns that unmet dependencies may cause issues", func() {
			_, logs, err := pack.Build.
				WithBuildpacks(nodeURI, npmURI).
				Execute(name, filepath.Join("testdata", "unmet_dep"))
			Expect(err).To(HaveOccurred())
			Expect(logs).To(ContainSubstring("vendored node_modules have unmet dependencies"))
			Expect(logs).To(ContainSubstring("npm list failed"))
			Expect(logs).To(ContainSubstring("npm ERR! missing: express@4.0.0, required by node_web_app@0.0.0"))
		})
	})
}
