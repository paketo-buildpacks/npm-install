package build_test

import (
	"github.com/buildpack/libbuildpack"
	"github.com/cloudfoundry/npm-cnb/internal/build"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Build", func() {

	Context("CreateLaunchMetadata", func() {

		It("returns launch metadata for running with npm", func() {
			Expect(build.CreateLaunchMetadata()).To(Equal(libbuildpack.LaunchMetadata{
				Processes: libbuildpack.Processes{
					libbuildpack.Process{
						Type:    "web",
						Command: "npm start",
					},
				},
			}))
		})
	})
})
