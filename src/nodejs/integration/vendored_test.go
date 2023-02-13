package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/switchblade"
	"github.com/sclevine/spec"

	. "github.com/cloudfoundry/switchblade/matchers"
	. "github.com/onsi/gomega"
)

func testVendored(platform switchblade.Platform, fixtures string) func(*testing.T, spec.G, spec.S) {
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

		it("uses vendored dependencies", func() {
			deployment, logs, err := platform.Deploy.
				WithEnv(map[string]string{"BP_DEBUG": "true"}).
				Execute(name, filepath.Join(fixtures, "vendored", "npm"))
			Expect(err).NotTo(HaveOccurred())

			Expect(logs).NotTo(ContainLines(
				MatchRegexp("PRO TIP:(.*) It is recommended to vendor the application's Node.js dependencies"),
			))

			checksumLines := ChecksumRegexp.FindAllStringSubmatch(logs.String(), -1)
			Expect(checksumLines).To(HaveLen(2))
			Expect(checksumLines[0][2]).To(Equal(checksumLines[1][2]), logs.String)

			Eventually(deployment).Should(Serve(ContainSubstring("0000000005")).WithEndpoint("/leftpad"))
		})

		context("when there are vendored binaries", func() {
			it("rebuilds those binaries", func() {
				deployment, _, err := platform.Deploy.
					WithEnv(map[string]string{"BP_DEBUG": "true"}).
					Execute(name, filepath.Join(fixtures, "vendored", "binaries"))
				Expect(err).NotTo(HaveOccurred())

				Eventually(deployment).Should(Serve(ContainSubstring("0000000005")).WithEndpoint("/leftpad"))
			})
		})

		context("with an app with a yarn.lock and vendored dependencies", func() {
			it("deploys without hitting the internet", func() {
				source := filepath.Join(fixtures, "vendored", "yarn")
				deployment, logs, err := platform.Deploy.
					WithEnv(map[string]string{"BP_DEBUG": "true"}).
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				Expect(filepath.Join(source, "node_modules")).To(BeADirectory())

				Eventually(deployment).Should(Serve(MatchRegexp("native time: \\d+\\.\\d+")).WithEndpoint("/microtime"))
				Expect(logs).To(ContainLines(
					ContainSubstring("Running yarn in offline mode"),
				))
			})
		})

		context("with an incomplete node_modules directory", func() {
			var source string

			it.Before(func() {
				var err error
				source, err = switchblade.Source(filepath.Join(fixtures, "vendored", "npm"))
				Expect(err).NotTo(HaveOccurred())

				Expect(os.RemoveAll(filepath.Join(source, "node_modules", "leftpad"))).To(Succeed())
			})

			it.After(func() {
				Expect(os.RemoveAll(source)).To(Succeed())
			})

			it("downloads missing dependencies from package.json", func() {
				deployment, _, err := platform.Deploy.
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				Expect(filepath.Join(source, "node_modules")).To(BeADirectory())
				Expect(filepath.Join(source, "node_modules", "hashish")).To(BeADirectory())
				Expect(filepath.Join(source, "node_modules", "leftpad")).NotTo(BeADirectory())

				Eventually(deployment).Should(Serve(ContainSubstring("0000000005")).WithEndpoint("/leftpad"))
			})
		})

		context("with an empty node_modules directory", func() {
			var source string

			it.Before(func() {
				var err error
				source, err = switchblade.Source(filepath.Join(fixtures, "vendored", "npm"))
				Expect(err).NotTo(HaveOccurred())

				Expect(os.RemoveAll(filepath.Join(source, "node_modules"))).To(Succeed())
				Expect(os.Mkdir(filepath.Join(source, "node_modules"), os.ModePerm)).To(Succeed())
			})

			it.After(func() {
				Expect(os.RemoveAll(source)).To(Succeed())
			})

			it("downloads missing dependencies from package.json", func() {
				deployment, logs, err := platform.Deploy.
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				Expect(logs).To(ContainLines(
					MatchRegexp("PRO TIP:(.*) It is recommended to vendor the application's Node.js dependencies"),
				))

				Expect(filepath.Join(source, "node_modules")).To(BeADirectory())
				Expect(filepath.Join(source, "node_modules", "leftpad")).ToNot(BeADirectory())
				Expect(filepath.Join(source, "node_modules", "hashish")).ToNot(BeADirectory())

				Eventually(deployment).Should(Serve(ContainSubstring("0000000005")).WithEndpoint("/leftpad"))
			})
		})

		context("with an incomplete package.json", func() {
			var source string

			it.Before(func() {
				var err error
				source, err = switchblade.Source(filepath.Join(fixtures, "vendored", "npm"))
				Expect(err).NotTo(HaveOccurred())

				var pkg map[string]interface{}
				content, err := os.ReadFile(filepath.Join(source, "package.json"))
				Expect(err).NotTo(HaveOccurred())
				Expect(json.Unmarshal(content, &pkg)).To(Succeed())

				pkg["dependencies"] = map[string]string{"leftpad": "~0.0.1"}
				content, err = json.Marshal(pkg)
				Expect(err).NotTo(HaveOccurred())

				Expect(os.WriteFile(filepath.Join(source, "package.json"), content, 0600)).To(Succeed())
			})

			it.After(func() {
				Expect(os.RemoveAll(source)).To(Succeed())
			})

			it("overwrites the vendored modules not listed in package.json", func() {
				deployment, _, err := platform.Deploy.
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				Expect(filepath.Join(source, "node_modules")).To(BeADirectory())
				Expect(filepath.Join(source, "node_modules", "leftpad")).To(BeADirectory())
				Expect(filepath.Join(source, "node_modules", "hashish")).To(BeADirectory())

				Eventually(deployment).Should(Serve(ContainSubstring("0000000005")).WithEndpoint("/leftpad"))
			})
		})
	}
}
