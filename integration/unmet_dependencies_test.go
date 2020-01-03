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
		it.Pend("warns that unmet dependencies may cause issues", func() {
			var err error
			app, err = dagger.NewPack(
				filepath.Join("testdata", "unmet_dep"),
				dagger.RandomImage(),
				dagger.SetBuildpacks(nodeURI, npmURI),
			).Build()
			Expect(err).ToNot(HaveOccurred())

			Expect(app.Start()).To(Succeed())

			Expect(app.BuildLogs()).To(ContainSubstring("Unmet dependencies don't fail npm install but may cause runtime issues"))
		})
	})
}
