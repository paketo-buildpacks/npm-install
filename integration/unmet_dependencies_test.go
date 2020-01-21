package integration_test

import (
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/dagger"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testUnmetDependencies(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
		app    *dagger.App
	)

	it.After(func() {
		Expect(app.Destroy()).To(Succeed())
	})

	context("when the package manager is npm", func() {
		it("warns that unmet dependencies may cause issues", func() {
			var err error
			app, err = dagger.NewPack(
				filepath.Join("testdata", "unmet_dep"),
				dagger.RandomImage(),
				dagger.SetBuildpacks(nodeURI, npmURI),
			).Build()

			Expect(err).To(MatchError(ContainSubstring("vendored node_modules have unmet dependencies")))
		})
	})
}
