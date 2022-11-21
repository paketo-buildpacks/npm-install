package npminstall_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	npminstall "github.com/paketo-buildpacks/npm-install"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/fs"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testUpdateNpmCache(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
		buf    *bytes.Buffer
		logger scribe.Emitter

		layersDir     string
		workingDir    string
		workingDirSum string
		cacheLayer    packit.Layer

		err error
	)

	it.Before(func() {
		buf = bytes.NewBuffer(nil)
		logger = scribe.NewEmitter(buf)

		layersDir, err = os.MkdirTemp("", "layers")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())
		Expect(os.MkdirAll(filepath.Join(workingDir, "npm-cache"), os.ModePerm)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(workingDir, "npm-cache", "some-file"), []byte("some-content"), os.ModePerm)).To(Succeed())

		workingDirSum, err = fs.NewChecksumCalculator().Sum(filepath.Join(workingDir, "npm-cache"))
		Expect(err).NotTo(HaveOccurred())

		layers := packit.Layers{Path: layersDir}
		cacheLayer, err = layers.Get("npm-cache")
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("UpdateNpmCacheLayer", func() {
		context("when cache layer is stale", func() {
			it("updates cache layer", func() {
				layer, err := npminstall.UpdateNpmCacheLayer(logger, workingDir, cacheLayer)
				Expect(err).NotTo(HaveOccurred())
				Expect(layer.Metadata).To(HaveKeyWithValue("cache_sha", workingDirSum))
				Expect(buf.String()).NotTo(ContainSubstring("Reusing cached layer"))

				Expect(filepath.Join(layer.Path, "some-file")).To(BeARegularFile())
			})
		})

		context("when cache layer is valid", func() {
			it.Before(func() {
				cacheLayer, err = cacheLayer.Reset()
				Expect(err).NotTo(HaveOccurred())
				cacheLayer.Metadata = map[string]interface{}{
					"cache_sha": workingDirSum,
				}
			})
			it("reuses the layer", func() {
				layer, err := npminstall.UpdateNpmCacheLayer(logger, workingDir, cacheLayer)
				Expect(err).NotTo(HaveOccurred())
				Expect(layer.Metadata).To(HaveKeyWithValue("cache_sha", workingDirSum))
				Expect(buf.String()).To(ContainSubstring("Reusing cached layer"))
			})
		})

		context("failure casees", func() {
			context("npm-cache is unreadable", func() {
				it.Before(func() {
					Expect(os.Chmod(workingDir, 0000)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Chmod(workingDir, os.ModePerm)).To(Succeed())
				})

				it("returns an error", func() {
					_, err = npminstall.UpdateNpmCacheLayer(logger, workingDir, cacheLayer)
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})
		})
	})
}
