package npminstall_test

import (
	"io/ioutil"
	"os"
	"testing"

	npminstall "github.com/paketo-buildpacks/npm-install"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testFileConcat(t *testing.T, context spec.G, it spec.S) {
	var Expect = NewWithT(t).Expect

	context("Concat", func() {
		var (
			path1      string
			path2      string
			fileConcat npminstall.FileConcat
		)

		it.Before(func() {
			file1, err := ioutil.TempFile("", "package.json")
			Expect(err).NotTo(HaveOccurred())
			defer file1.Close()

			_, err = file1.WriteString(`{
				"engines": {
					"node": "1.2.3"
				}
			}`)
			Expect(err).NotTo(HaveOccurred())

			file2, err := ioutil.TempFile("", "package-lock.json")
			Expect(err).NotTo(HaveOccurred())
			defer file2.Close()

			_, err = file2.WriteString(`{
				"engines": {
					"some-other-node": "1.2.3.4"
				}
			}`)
			Expect(err).NotTo(HaveOccurred())

			path1 = file1.Name()
			path2 = file2.Name()

			fileConcat = npminstall.NewFileConcat()
		})

		it.After(func() {
			Expect(os.RemoveAll(path1)).To(Succeed())
			Expect(os.RemoveAll(path2)).To(Succeed())
		})

		it("concatenates the two files and write them to a temp file", func() {
			outputFile, err := fileConcat.Concat(path1, path2)
			Expect(err).NotTo(HaveOccurred())
			defer func() {
				Expect(os.Remove(outputFile)).NotTo(HaveOccurred())
			}()
			Expect(outputFile).Should(BeARegularFile())

			contents, err := ioutil.ReadFile(outputFile)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(contents)).To(Equal(`{
				"engines": {
					"node": "1.2.3"
				}
			}{
				"engines": {
					"some-other-node": "1.2.3.4"
				}
			}`))
		})

		context("failure cases", func() {
			context("an input file can not be read", func() {
				it.Before(func() {
					Expect(os.Chmod(path1, 0000)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := fileConcat.Concat(path1, path2)
					Expect(err).To(MatchError(ContainSubstring("could not read file: ")))
				})
			})
		})
	})
}
