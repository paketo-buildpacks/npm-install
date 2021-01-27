package npminstall_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	npminstall "github.com/paketo-buildpacks/npm-install"
	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/scribe"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testEnvironment(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layer packit.Layer
		path  string

		buffer      *bytes.Buffer
		environment npminstall.Environment
	)

	it.Before(func() {
		var err error
		path, err = ioutil.TempDir("", "layer-dir")
		Expect(err).NotTo(HaveOccurred())

		layer = packit.Layer{Path: path}

		layer, err = layer.Reset()
		Expect(err).NotTo(HaveOccurred())

		buffer = bytes.NewBuffer(nil)
		environment = npminstall.NewEnvironment(scribe.NewLogger(buffer))
	})

	it.After(func() {
		Expect(os.RemoveAll(path)).To(Succeed())
	})

	context("Configure", func() {
		it("configures the environment variables", func() {
			err := environment.Configure(layer)
			Expect(err).NotTo(HaveOccurred())

			Expect(layer.LaunchEnv).To(Equal(packit.Environment{
				"NPM_CONFIG_LOGLEVEL.default": "error",
			}))

			Expect(layer.SharedEnv).To(Equal(packit.Environment{
				"PATH.append": filepath.Join(layer.Path, "node_modules", ".bin"),
				"PATH.delim":  string(os.PathListSeparator),
			}))
		})

		context("when NPM_CONFIG_LOGLEVEL is set", func() {
			it.Before(func() {
				os.Setenv("NPM_CONFIG_LOGLEVEL", "some-val")
			})

			it.After(func() {
				os.Unsetenv("NPM_CONFIG_LOGLEVEL")
			})

			it("configures variables using given value", func() {
				err := environment.Configure(layer)
				Expect(err).NotTo(HaveOccurred())

				Expect(layer.LaunchEnv).To(Equal(packit.Environment{
					"NPM_CONFIG_LOGLEVEL.default": "some-val",
				}))
			})
		})

	})
}
