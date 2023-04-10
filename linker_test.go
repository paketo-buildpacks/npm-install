package npminstall_test

import (
	"os"
	"path/filepath"
	"testing"

	npminstall "github.com/paketo-buildpacks/npm-install"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testLinker(t *testing.T, context spec.G, it spec.S) {
	var Expect = NewWithT(t).Expect

	var (
		sourceDir, targetDir, tmpDir string
		source, target               string
		linker                       npminstall.Linker
	)

	it.Before(func() {
		var err error
		sourceDir, err = os.MkdirTemp("", "source")
		Expect(err).NotTo(HaveOccurred())

		source = filepath.Join(sourceDir, "file")

		file, err := os.Create(source)
		Expect(err).NotTo(HaveOccurred())
		Expect(file.Close()).To(Succeed())

		targetDir, err = os.MkdirTemp("", "target")
		Expect(err).NotTo(HaveOccurred())

		target = filepath.Join(targetDir, "file")

		tmpDir, err = os.MkdirTemp("", "tmp")
		Expect(err).NotTo(HaveOccurred())

		linker = npminstall.NewLinker(tmpDir)
	})

	it.After(func() {
		Expect(os.RemoveAll(sourceDir)).To(Succeed())
		Expect(os.RemoveAll(targetDir)).To(Succeed())
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	context("Link", func() {
		it("links the source to the target through an indirection path in a temporary directory", func() {
			err := linker.Link(source, target)
			Expect(err).NotTo(HaveOccurred())

			link, err := os.Readlink(source)
			Expect(err).NotTo(HaveOccurred())

			Expect(link).To(HavePrefix(tmpDir))

			link, err = os.Readlink(link)
			Expect(err).NotTo(HaveOccurred())

			Expect(link).To(Equal(target))
		})

		context("failure cases", func() {
			context("when the source cannot be removed", func() {
				it.Before(func() {
					Expect(os.Chmod(sourceDir, 0000)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Chmod(sourceDir, os.ModePerm)).To(Succeed())
				})

				it("returns an error", func() {
					err := linker.Link(source, target)
					Expect(err).To(MatchError(ContainSubstring("failed to remove link source:")))
				})
			})

			context("when the indirection link cannot be removed", func() {
				it.Before(func() {
					Expect(os.Chmod(tmpDir, 0000)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Chmod(tmpDir, os.ModePerm)).To(Succeed())
				})

				it("returns an error", func() {
					err := linker.Link(source, target)
					Expect(err).To(MatchError(ContainSubstring("failed to remove link indirection:")))
				})
			})
		})
	})

	context("WithPath", func() {
		it("places the symlink at the given path within the indirection directory", func() {
			err := linker.WithPath("some/link/path").Link(source, target)
			Expect(err).NotTo(HaveOccurred())

			link, err := os.Readlink(source)
			Expect(err).NotTo(HaveOccurred())

			Expect(link).To(Equal(filepath.Join(tmpDir, "some", "link", "path")))

			link, err = os.Readlink(link)
			Expect(err).NotTo(HaveOccurred())

			Expect(link).To(Equal(target))
		})
	})
}
