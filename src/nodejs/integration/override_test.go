package integration_test

import (
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/switchblade"
	"github.com/sclevine/spec"

	. "github.com/cloudfoundry/switchblade/matchers"
	. "github.com/onsi/gomega"
)

func testOverride(platform switchblade.Platform, fixtures string) func(*testing.T, spec.G, spec.S) {
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

		it("installs node from override buildpack", func() {
			_, logs, err := platform.Deploy.
				WithBuildpacks(
					"override_buildpack",
					"nodejs_buildpack",
				).
				Execute(name, filepath.Join(fixtures, "simple"))
			Expect(err).To(HaveOccurred())

			Expect(logs.String()).To(SatisfyAll(
				ContainLines(ContainSubstring("-----> OverrideYML Buildpack")),
				ContainLines(ContainSubstring("-----> Installing node")),
				ContainLines(MatchRegexp("Copy .*/node.tgz")),
				ContainLines(ContainSubstring("Unable to install node: dependency sha256 mismatch: expected sha256 062d906c87839d03b243e2821e10653c89b4c92878bfe2bf995dec231e117bfc, actual sha256 b56b58ac21f9f42d032e1e4b8bf8b8823e69af5411caa15aee2b140bc756962f")),
			))
		})
	}
}
