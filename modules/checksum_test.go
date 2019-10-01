package modules_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/cloudfoundry/npm-cnb/modules"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testChecksum(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	when("String", func() {
		it("returns a string representation of the checksum", func() {
			checksum := modules.NewChecksum(strings.NewReader("some-content"))

			sum, err := checksum.String()
			Expect(err).NotTo(HaveOccurred())
			Expect(sum).To(Equal("0a8cac771ca188eacc57e2c96c31f5611925c5ecedccb16b8c236d6c0d325112"))
		})

		when("the reader cannot be copied", func() {
			it("returns an error", func() {
				checksum := modules.NewChecksum(errReader{errors.New("failed to read")})

				_, err := checksum.String()
				Expect(err).To(MatchError("failed to calculate checksum: failed to read"))
			})
		})
	})

	when("NewTimeChecksum", func() {
		it("calculates the checksum from a timestamp", func() {
			now, err := time.Parse(time.RFC3339, "2006-01-02T15:04:05Z")
			Expect(err).NotTo(HaveOccurred())

			checksum := modules.NewTimeChecksum(now)

			sum, err := checksum.String()
			Expect(err).NotTo(HaveOccurred())
			Expect(sum).To(Equal("b3cf032436957e22a4c99036760a59c65bd4bdd981f3d2f02e4d3a80a0cf0cfe"))
		})
	})
}
