package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testUnmetDependencies(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		pack   occam.Pack
		docker occam.Docker

		name   string
		source string
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
		Expect(os.RemoveAll(source)).To(Succeed())
	})

	context("when the package manager is npm", func() {
		it("warns that unmet dependencies may cause issues", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "unmet_dep"))
			Expect(err).NotTo(HaveOccurred())

			_, logs, err := pack.WithNoColor().Build.
				WithPullPolicy("never").
				WithBuildpacks(
					nodeURI,
					buildpackURI,
					buildPlanURI,
				).
				Execute(name, source)
			Expect(err).To(HaveOccurred())
			Expect(logs.String()).To(ContainSubstring("vendored node_modules have unmet dependencies"))
			Expect(logs.String()).To(ContainSubstring("npm list failed"))
			Expect(logs.String()).To(ContainSubstring("UNMET DEPENDENCY express@4.17.1"))
		})
	})
}
