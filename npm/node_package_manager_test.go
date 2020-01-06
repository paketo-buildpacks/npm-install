package npm_test

import (
	"errors"
	"testing"

	"github.com/cloudfoundry/npm-cnb/npm"
	"github.com/cloudfoundry/npm-cnb/npm/fakes"
	"github.com/cloudfoundry/packit/pexec"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testNodePackageManager(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		executable         *fakes.Executable
		nodePackageManager npm.NodePackageManager
	)

	it.Before(func() {
		executable = &fakes.Executable{}

		nodePackageManager = npm.NewNodePackageManager(executable)
	})

	context("Install", func() {
		it("executes the npm install command", func() {
			err := nodePackageManager.Install("working-dir")
			Expect(err).NotTo(HaveOccurred())

			Expect(executable.ExecuteCall.Receives.Execution).To(Equal(pexec.Execution{
				Args: []string{"install"},
				Dir:  "working-dir",
			}))
		})

		context("failure cases", func() {
			context("when the executable fails", func() {
				it.Before(func() {
					executable.ExecuteCall.Returns.Err = errors.New("failed to execute")
				})

				it("returns an error", func() {
					err := nodePackageManager.Install("working-dir")
					Expect(err).To(MatchError("failed to execute"))
				})
			})
		})
	})
}
