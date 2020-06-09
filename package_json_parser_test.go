package npm_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/paketo-buildpacks/npm"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testPackageJSONParser(t *testing.T, context spec.G, it spec.S) {
	var Expect = NewWithT(t).Expect

	context("ParseVersion", func() {
		var (
			path   string
			parser npm.PackageJSONParser
		)

		it.Before(func() {
			file, err := ioutil.TempFile("", "package.json")
			Expect(err).NotTo(HaveOccurred())
			defer file.Close()

			_, err = file.WriteString(`{
				"engines": {
					"node": "1.2.3"
				}
			}`)
			Expect(err).NotTo(HaveOccurred())

			path = file.Name()

			parser = npm.NewPackageJSONParser()
		})

		it.After(func() {
			Expect(os.RemoveAll(path)).To(Succeed())
		})

		it("parses the node engine version from a package.json file", func() {
			version, err := parser.ParseVersion(path)
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal("1.2.3"))
		})

		context("failure cases", func() {
			context("when the package.json file does not exist", func() {
				it("returns an error", func() {
					_, err := parser.ParseVersion("/missing/file")
					Expect(err).To(MatchError(ContainSubstring("no such file or directory")))
				})
			})

			context("when the package.json contents are malformed", func() {
				it.Before(func() {
					err := ioutil.WriteFile(path, []byte("%%%"), 0644)
					Expect(err).NotTo(HaveOccurred())
				})

				it("returns an error", func() {
					_, err := parser.ParseVersion(path)
					Expect(err).To(MatchError(ContainSubstring("invalid character")))
				})
			})
		})
	})
}
