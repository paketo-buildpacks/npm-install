package detection_test

import (
	"testing"

	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/npm-cnb/detection"
	"github.com/cloudfoundry/npm-cnb/modules"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testPlan(t *testing.T, when spec.G, it spec.S) {
	when("Plan", func() {
		it.Before(func() {
			RegisterTestingT(t)
		})

		it("returns a plan for the node modules dependency", func() {
			plan := detection.NewPlan("1.2.3")
			Expect(plan).To(Equal(buildplan.Plan{
				Provides: []buildplan.Provided{
					{Name: modules.Dependency},
				},
				Requires: []buildplan.Required{
					{
						Name:    modules.NodeDependency,
						Version: "1.2.3",
						Metadata: buildplan.Metadata{
							"build":          true,
							"launch":         true,
							"version-source": "package.json",
						},
					},
					{
						Name: modules.Dependency,
						Metadata: buildplan.Metadata{
							"launch": true,
						},
					},
				},
			}))
		})

		when("the version number is empty", func() {
			it("returns a plan with an empty version-source", func() {
				plan := detection.NewPlan("")
				Expect(plan).To(Equal(buildplan.Plan{
					Provides: []buildplan.Provided{
						{Name: modules.Dependency},
					},
					Requires: []buildplan.Required{
						{
							Name: modules.NodeDependency,
							Metadata: buildplan.Metadata{
								"build":          true,
								"launch":         true,
								"version-source": "",
							},
						},
						{
							Name: modules.Dependency,
							Metadata: buildplan.Metadata{
								"launch": true,
							},
						},
					},
				}))
			})
		})
	})
}
