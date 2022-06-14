package integration_test

import (
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/switchblade"
	"github.com/sclevine/spec"

	. "github.com/cloudfoundry/switchblade/matchers"
	. "github.com/onsi/gomega"
)

func testOffline(platform switchblade.Platform, fixtures string) func(*testing.T, spec.G, spec.S) {
	return func(t *testing.T, context spec.G, it spec.S) {
		var (
			Expect     = NewWithT(t).Expect
			Eventually = NewWithT(t).Eventually

			name string
		)

		it.Before(func() {
			var err error
			name, err = switchblade.RandomName()
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(platform.Delete.Execute(name)).To(Succeed())
		})

		it("builds npm apps without internet access", func() {
			deployment, _, err := platform.Deploy.
				WithoutInternetAccess().
				Execute(name, filepath.Join(fixtures, "vendored", "npm"))
			Expect(err).NotTo(HaveOccurred())

			Eventually(deployment).Should(Serve("Hello, World!"))
		})

		it("builds yarn apps without internet access", func() {
			deployment, _, err := platform.Deploy.
				WithoutInternetAccess().
				Execute(name, filepath.Join(fixtures, "vendored", "yarn"))
			Expect(err).NotTo(HaveOccurred())

			Eventually(deployment).Should(Serve("Hello, World!"))
		})
	}
}
