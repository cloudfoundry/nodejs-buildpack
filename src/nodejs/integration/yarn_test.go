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

func testYarn(platform switchblade.Platform, fixtures string) func(*testing.T, spec.G, spec.S) {
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

			source, err = switchblade.Source(filepath.Join(fixtures, "yarn", "simple"))
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(platform.Delete.Execute(name)).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		it("successfully deploys and vendors the dependencies via yarn", func() {
			deployment, logs, err := platform.Deploy.
				WithEnv(map[string]string{"BP_DEBUG": "true"}).
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred())

			Expect(filepath.Join(source, "node_modules")).ToNot(BeADirectory())

			checksumLines := ChecksumRegexp.FindAllStringSubmatch(logs.String(), -1)
			Expect(checksumLines).To(HaveLen(2))
			Expect(checksumLines[0][2]).To(Equal(checksumLines[1][2]))

			Expect(logs).To(ContainLines(
				ContainSubstring("Running yarn in online mode"),
			))

			Eventually(deployment).Should(Serve(ContainSubstring("Hello, World!")))
		})

		context("with an app with an out of date yarn.lock", func() {
			it.Before(func() {
				var pkg map[string]interface{}
				content, err := os.ReadFile(filepath.Join(source, "package.json"))
				Expect(err).NotTo(HaveOccurred())
				Expect(json.Unmarshal(content, &pkg)).To(Succeed())

				dependencies, ok := pkg["dependencies"].(map[string]interface{})
				Expect(ok).To(BeTrue())

				dependencies["leftpad"] = "~0.0.1"
				pkg["dependencies"] = dependencies
				content, err = json.Marshal(pkg)
				Expect(err).NotTo(HaveOccurred())

				Expect(os.WriteFile(filepath.Join(source, "package.json"), content, 0600)).To(Succeed())
			})
		})

		context("when there are unmet dependencies", func() {
			it.Before(func() {
				var pkg map[string]interface{}
				content, err := os.ReadFile(filepath.Join(source, "package.json"))
				Expect(err).NotTo(HaveOccurred())
				Expect(json.Unmarshal(content, &pkg)).To(Succeed())

				dependencies, ok := pkg["dependencies"].(map[string]interface{})
				Expect(ok).To(BeTrue())

				dependencies["grunt-steroids"] = "0.2.3"
				pkg["dependencies"] = dependencies
				content, err = json.Marshal(pkg)
				Expect(err).NotTo(HaveOccurred())

				Expect(os.WriteFile(filepath.Join(source, "package.json"), content, 0600)).To(Succeed())
			})

			it("prints a warning", func() {
				_, logs, err := platform.Deploy.Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				Expect(logs).To(ContainLines(
					ContainSubstring("Unmet dependencies don't fail yarn install but may cause runtime issues"),
				))
			})
		})

		context("deploying a Node.js app that uses yarn workspaces", func() {
			it.Before(func() {
				var err error
				source, err = switchblade.Source(filepath.Join(fixtures, "yarn", "workspaces"))
				Expect(err).NotTo(HaveOccurred())
			})

			it("outputs config contents when queried", func() {
				deployment, _, err := platform.Deploy.Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				Eventually(deployment).Should(Serve(
					ContainSubstring(`"config":{"prop1":"Package A value 1","prop2":"Package A value 2"}`),
				).WithEndpoint("/check"))
			})
		})
	}
}
