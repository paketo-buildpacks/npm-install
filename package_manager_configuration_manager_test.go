package npminstall_test

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"

	npminstall "github.com/paketo-buildpacks/npm-install"
	"github.com/paketo-buildpacks/npm-install/fakes"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	"github.com/paketo-buildpacks/packit/v2/servicebindings"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testPackageManagerConfigurationManager(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		buffer          *bytes.Buffer
		bindingResolver *fakes.BindingResolver

		packageManagerConfigurationManager npminstall.PackageManagerConfigurationManager
	)

	it.Before(func() {
		bindingResolver = &fakes.BindingResolver{}

		buffer = bytes.NewBuffer(nil)

		packageManagerConfigurationManager = npminstall.NewPackageManagerConfigurationManager(bindingResolver, scribe.NewEmitter(buffer))
	})

	context("DeterminePath", func() {
		context("when there is a configuration binding set", func() {
			it.Before(func() {
				bindingResolver.ResolveCall.Returns.BindingSlice = []servicebindings.Binding{
					{
						Name: "first",
						Type: "some-typ",
						Path: "some-binding-path",
						Entries: map[string]*servicebindings.Entry{
							"some-entry": servicebindings.NewEntry("some-path"),
						},
					},
				}
			})
			it("returns a path to the configuration file", func() {
				path, err := packageManagerConfigurationManager.DeterminePath("some-typ", "platform-dir", "some-entry")
				Expect(err).NotTo(HaveOccurred())

				Expect(path).To(Equal(filepath.Join("some-binding-path", "some-entry")))

				Expect(bindingResolver.ResolveCall.Receives.Typ).To(Equal("some-typ"))
				Expect(bindingResolver.ResolveCall.Receives.PlatformDir).To(Equal("platform-dir"))

				Expect(buffer.String()).To(ContainSubstring("Loading service binding of type 'some-typ'"))
			})
		})

		context("failure cases", func() {
			context("when the binding resolver fails", func() {
				it.Before(func() {
					bindingResolver.ResolveCall.Returns.Error = errors.New("failed to resolve binding")
				})
				it("returns an error", func() {
					_, err := packageManagerConfigurationManager.DeterminePath("some-typ", "platform-dir", "some-entry")
					Expect(err).To(MatchError("failed to resolve binding"))
				})
			})

			context("when more than one binding is found", func() {
				it.Before(func() {
					bindingResolver.ResolveCall.Returns.BindingSlice = []servicebindings.Binding{
						{
							Name: "first",
							Type: "some-typ",
							Path: "some-binding-path",
							Entries: map[string]*servicebindings.Entry{
								"some-entry": servicebindings.NewEntry("some-path"),
							},
						},
						{
							Name: "second",
							Type: "some-typ",
							Path: "some-binding-path",
							Entries: map[string]*servicebindings.Entry{
								"another-entry": servicebindings.NewEntry("some-path"),
							},
						},
					}
				})
				it("returns an error", func() {
					_, err := packageManagerConfigurationManager.DeterminePath("some-typ", "platform-dir", "some-entry")
					Expect(err).To(MatchError("failed: binding resolver found more than one binding of type 'some-typ'"))
				})
			})

			context("when the binding is missing the required entry", func() {
				it.Before(func() {
					bindingResolver.ResolveCall.Returns.BindingSlice = []servicebindings.Binding{
						{
							Name: "first",
							Type: "some-typ",
							Path: "some-binding-path",
							Entries: map[string]*servicebindings.Entry{
								"other-entry": servicebindings.NewEntry("some-path"),
							},
						},
					}
				})
				it("returns an error", func() {
					_, err := packageManagerConfigurationManager.DeterminePath("some-typ", "platform-dir", "some-entry")
					Expect(err).To(MatchError("failed: binding of type 'some-typ' does not contain required entry 'some-entry'"))
				})
			})
		})
	})
}
