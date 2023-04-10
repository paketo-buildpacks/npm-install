package npminstall_test

import (
	"os"
	"path/filepath"
	"testing"

	npminstall "github.com/paketo-buildpacks/npm-install"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testEnvironment(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		tmpDir string
		path   string

		environment npminstall.Environment
	)

	it.Before(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "")
		Expect(err).NotTo(HaveOccurred())

		path = filepath.Join(tmpDir, "buildpack.toml")

		Expect(os.WriteFile(path, []byte(`
[metadata]
	[[metadata.configurations]]
		description = "variable with a value"
		name = "SOME_KEY"

	[[metadata.configurations]]
		description = "variable with a boolean value"
		name = "BOOL_KEY"

	[[metadata.configurations]]
		description = "variable with no value"
		name = "UNSET_KEY"

	[[metadata.configurations]]
		default = "default-value"
		description = "variable with a default"
		name = "DEFAULT_KEY"
`), 0600)).To(Succeed())

		environment, err = npminstall.ParseEnvironment(path, []string{
			"SOME_KEY=some-value",
			"BOOL_KEY=true",
		})
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	context("ParseEnvironment", func() {
		context("failure cases", func() {
			context("when the buildpack.toml cannot be read", func() {
				it("returns an error", func() {
					_, err := npminstall.ParseEnvironment("/no/such/path/to/buildpack.toml", nil)
					Expect(err).To(MatchError(ContainSubstring("failed to read \"buildpack.toml\"")))
				})
			})

			context("when the buildpack.toml cannot be parsed", func() {
				it.Before(func() {
					Expect(os.WriteFile(path, []byte("%%%"), 0600)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := npminstall.ParseEnvironment(path, nil)
					Expect(err).To(MatchError(ContainSubstring("failed to parse \"buildpack.toml\"")))
				})
			})
		})
	})

	context("Lookup", func() {
		it("finds the value of an environment variable", func() {
			value, ok := environment.Lookup("SOME_KEY")
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal("some-value"))
		})

		context("when the variable is not configured", func() {
			it("reports not found", func() {
				value, ok := environment.Lookup("NO_SUCH_KEY")
				Expect(ok).To(BeFalse())
				Expect(value).To(BeEmpty())
			})
		})

		context("when the environment variable is not present", func() {
			it("reports not found", func() {
				value, ok := environment.Lookup("UNSET_KEY")
				Expect(ok).To(BeFalse())
				Expect(value).To(BeEmpty())
			})

			context("when there is a default defined in the buildpack.toml", func() {
				it("returns that default", func() {
					value, ok := environment.Lookup("DEFAULT_KEY")
					Expect(ok).To(BeTrue())
					Expect(value).To(Equal("default-value"))
				})
			})
		})
	})

	context("LookupBool", func() {
		it("finds the value of a boolean environment variable", func() {
			found, err := environment.LookupBool("BOOL_KEY")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
		})

		context("when the environment variable is not present", func() {
			it("reports not found", func() {
				found, err := environment.LookupBool("NO_SUCH_KEY")
				Expect(err).NotTo(HaveOccurred())
				Expect(found).To(BeFalse())
			})
		})

		context("failure cases", func() {
			context("when the value cannot be parsed to boolean", func() {
				it("returns an error", func() {
					_, err := environment.LookupBool("SOME_KEY")
					Expect(err).To(MatchError(ContainSubstring(`failed to parse boolean environment variable "SOME_KEY"`)))
					Expect(err).To(MatchError(ContainSubstring(`parsing "some-value": invalid syntax`)))
				})
			})
		})
	})
}
