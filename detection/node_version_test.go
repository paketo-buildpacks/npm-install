package detection_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/cloudfoundry/npm-cnb/detection"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testNodeVersion(t *testing.T, when spec.G, it spec.S) {
	when("GetNodeVersion", func() {
		var path string

		it.Before(func() {
			RegisterTestingT(t)

			file, err := ioutil.TempFile("", "package.json")
			Expect(err).NotTo(HaveOccurred())
			defer file.Close()

			path = file.Name()

			_, err = file.WriteString(`{
				"engines": {
					"node": "1.2.3"
				}
			}`)
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(os.Remove(path)).To(Succeed())
		})

		it("parses the node engine version out of a package.json file", func() {
			version, err := detection.GetNodeVersion(path)
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal("1.2.3"))
		})

		when("the package.json file does not exist", func() {
			it("returns an error", func() {
				_, err := detection.GetNodeVersion("/no/such/path/package.json")
				Expect(err).To(MatchError(ContainSubstring("no such file or directory")))
			})
		})

		when("the package.json is malformed", func() {
			it.Before(func() {
				err := ioutil.WriteFile(path, []byte("%%%"), 0644)
				Expect(err).NotTo(HaveOccurred())
			})

			it("returns an error", func() {
				_, err := detection.GetNodeVersion(path)
				Expect(err).To(MatchError(ContainSubstring("unable to parse package.json")))
				Expect(err).To(MatchError(ContainSubstring("invalid character '%'")))
			})
		})
	})
}
