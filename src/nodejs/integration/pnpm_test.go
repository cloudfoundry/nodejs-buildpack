package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/switchblade"
	"github.com/sclevine/spec"

	. "github.com/cloudfoundry/switchblade/matchers"
	. "github.com/onsi/gomega"
)

func testPNPM(platform switchblade.Platform, fixtures string) func(*testing.T, spec.G, spec.S) {
	return func(t *testing.T, context spec.G, it spec.S) {
		var (
			Expect     = NewWithT(t).Expect
			Eventually = NewWithT(t).Eventually

			name   string
			source string
		)

		it.Before(func() {
			var err error
			name, err = switchblade.RandomName()
			Expect(err).NotTo(HaveOccurred())

			source, err = switchblade.Source(filepath.Join(fixtures, "pnpm", "simple"))
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(platform.Delete.Execute(name)).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		it("successfully deploys and vendors the dependencies via pnpm", func() {
			deployment, logs, err := platform.Deploy.
				WithEnv(map[string]string{"BP_DEBUG": "true"}).
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred())

			Expect(filepath.Join(source, "node_modules")).ToNot(BeADirectory())

			Expect(logs).To(ContainLines(
				ContainSubstring("Installing node modules (pnpm-lock.yaml)"),
			))

			// Verify store directory is used
			Expect(logs).To(ContainLines(
				ContainSubstring("Using pnpm store directory:"),
			))

			Eventually(deployment).Should(Serve(ContainSubstring("Hello, World!")))
		})

		context("deploying a Node.js app that uses pnpm workspaces", func() {
			it.Before(func() {
				var err error
				source, err = switchblade.Source(filepath.Join(fixtures, "pnpm", "workspaces"))
				Expect(err).NotTo(HaveOccurred())
			})

			it("successfully deploys and vendors the dependencies via pnpm workspaces", func() {
				deployment, logs, err := platform.Deploy.
					WithEnv(map[string]string{"BP_DEBUG": "true"}).
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				Expect(filepath.Join(source, "node_modules")).ToNot(BeADirectory())

				Expect(logs).To(ContainLines(
					ContainSubstring("Installing node modules (pnpm-lock.yaml)"),
				))

				Eventually(deployment).Should(Serve(ContainSubstring("Hello from Workspace! Hello from sample-lib")))
			})
		})

		context("when there are unmet dependencies", func() {
			it.Before(func() {
				var err error
				source, err = switchblade.Source(filepath.Join(fixtures, "pnpm", "unmet"))
				Expect(err).NotTo(HaveOccurred())
			})

			it("prints a warning", func() {
				_, logs, err := platform.Deploy.
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				Expect(logs).To(ContainLines(
					ContainSubstring("Unmet dependencies don't fail pnpm install but may cause runtime issues"),
				))
			})
		})
	}
}
