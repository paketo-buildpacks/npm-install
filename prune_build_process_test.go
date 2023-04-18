package npminstall_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"testing"

	npminstall "github.com/paketo-buildpacks/npm-install"
	"github.com/paketo-buildpacks/npm-install/fakes"
	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testPruneBuildProcess(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		modulesDir  string
		cacheDir    string
		workingDir  string
		executable  *fakes.Executable
		environment *fakes.EnvironmentConfig
		buffer      *bytes.Buffer

		process npminstall.PruneBuildProcess
	)

	it.Before(func() {
		var err error
		modulesDir, err = os.MkdirTemp("", "modules")
		Expect(err).NotTo(HaveOccurred())

		cacheDir, err = os.MkdirTemp("", "cache")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		executable = &fakes.Executable{}
		executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
			fmt.Fprintln(execution.Stdout, "stdout output")
			fmt.Fprintln(execution.Stderr, "stderr output")
			return nil
		}

		environment = &fakes.EnvironmentConfig{}
		environment.LookupCall.Returns.Value = "some-val"
		environment.LookupCall.Returns.Found = true

		buffer = bytes.NewBuffer(nil)

		process = npminstall.NewPruneBuildProcess(executable, environment, scribe.NewLogger(buffer))
	})

	it.After(func() {
		Expect(os.RemoveAll(modulesDir)).To(Succeed())
		Expect(os.RemoveAll(cacheDir)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("ShouldRun", func() {
		it("returns true", func() {
			run, sha, err := process.ShouldRun(workingDir, nil, "some-npmrc-path")
			Expect(err).NotTo(HaveOccurred())
			Expect(run).To(BeTrue())
			Expect(sha).To(BeEmpty())
		})
	})

	context("Run", func() {
		it("succeeds", func() {
			Expect(process.Run(modulesDir, cacheDir, workingDir, "some-npmrc-path", true)).To(Succeed())
			Expect(executable.ExecuteCall.Receives.Execution.Args).To(Equal([]string{"prune"}))
			Expect(executable.ExecuteCall.Receives.Execution.Dir).To(Equal(workingDir))
			Expect(executable.ExecuteCall.Receives.Execution.Env).To(Equal(append(os.Environ(), "NPM_CONFIG_LOGLEVEL=some-val", "NPM_CONFIG_GLOBALCONFIG=some-npmrc-path")))
			Expect(buffer.String()).To(ContainLines(
				"    Running 'npm prune'",
				"      stdout output",
				"      stderr output",
			))
		})

		context("failure cases", func() {
			context("when the executable fails", func() {
				it.Before(func() {
					executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
						fmt.Fprintln(execution.Stdout, "install error on stdout")
						fmt.Fprintln(execution.Stderr, "install error on stderr")
						return errors.New("failed to execute")
					}
				})

				it("returns an error", func() {
					err := process.Run(modulesDir, cacheDir, workingDir, "", true)
					Expect(buffer.String()).To(ContainLines(
						"    Running 'npm prune'",
						"      install error on stdout",
						"      install error on stderr",
					))
					Expect(err).To(MatchError("npm install failed: failed to execute"))
				})
			})
		})
	})
}
