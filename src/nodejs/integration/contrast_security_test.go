package integration_test

import (
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/switchblade"
	"github.com/sclevine/spec"

	. "github.com/cloudfoundry/switchblade/matchers"
	. "github.com/onsi/gomega"
)

func testContrastSecurity(platform switchblade.Platform, fixtures string) func(*testing.T, spec.G, spec.S) {
	return func(t *testing.T, context spec.G, it spec.S) {
		var (
			Expect = NewWithT(t).Expect

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

		it("uses the Contrast Security service binding", func() {
			_, logs, err := platform.Deploy.
				WithServices(map[string]switchblade.Service{
					"some-contrast-security": {
						"api_key":        "sample_api_key",
						"org_uuid":       "sample_org_uuid",
						"service_key":    "sample_service_key",
						"teamserver_url": "sample_teamserver_url",
						"username":       "sample_username",
					},
				}).
				Execute(name, filepath.Join(fixtures, "simple"))
			Expect(err).NotTo(HaveOccurred())

			Expect(logs).To(ContainLines(
				ContainSubstring("Contrast Security credentials found"),
			))
		})
	}
}
