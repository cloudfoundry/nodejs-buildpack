package integration_test

import (
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/switchblade"
	"github.com/sclevine/spec"

	. "github.com/cloudfoundry/switchblade/matchers"
	. "github.com/onsi/gomega"
)

func testMultibuildpack(platform switchblade.Platform, fixtures string) func(*testing.T, spec.G, spec.S) {
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

		it("finds the supplied dependency in the runtime container", func() {
			deployment, logs, err := platform.Deploy.
				WithBuildpacks(
					"https://github.com/cloudfoundry/dotnet-core-buildpack#master",
					"nodejs_buildpack",
				).
				Execute(name, filepath.Join(fixtures, "simple"))
			Expect(err).NotTo(HaveOccurred())

			Expect(logs).To(ContainLines(
				ContainSubstring("Supplying Dotnet Core"),
			))

			Eventually(deployment).Should(Serve(MatchRegexp(`dotnet: \d+\.\d+\.\d+`)).WithEndpoint("/dotnet"))
		})
	}
}
