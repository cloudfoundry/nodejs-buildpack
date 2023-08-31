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

func testNPM(platform switchblade.Platform, fixtures string) func(*testing.T, spec.G, spec.S) {
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

		it("successfully deploys and vendors the dependencies", func() {
			source := filepath.Join(fixtures, "npm")
			Expect(filepath.Join(source, "node_modules")).NotTo(BeADirectory())

			deployment, logs, err := platform.Deploy.Execute(name, source)
			Expect(err).NotTo(HaveOccurred())

			Expect(logs.String()).To(SatisfyAll(
				MatchRegexp("PRO TIP:(.*) It is recommended to vendor the application's Node.js dependencies"),
				ContainSubstring("Current dir: /tmp/app"),
				ContainSubstring("Running heroku-prebuild (npm)"),
				ContainSubstring("Running heroku-postbuild (npm)"),
			))

			Eventually(deployment).Should(Serve("Hello, World!"))

			Eventually(deployment).Should(Serve(SatisfyAll(
				ContainSubstring("Text: Hello Buildpacks Team"),
				ContainSubstring("Text: Goodbye Buildpacks Team"),
			)).WithEndpoint("/prepost"))

			Eventually(deployment).Should(Serve(ContainSubstring("Successfully created mysql client")).WithEndpoint("/mysql"))
		})

		context("when a specific npm version is specified in the package.json", func() {
			var source string

			it.Before(func() {
				var err error
				source, err = switchblade.Source(filepath.Join(fixtures, "npm"))
				Expect(err).NotTo(HaveOccurred())

				file, err := os.OpenFile(filepath.Join(source, "package.json"), os.O_RDWR, 0600)
				Expect(err).NotTo(HaveOccurred())

				var pkg map[string]interface{}
				Expect(json.NewDecoder(file).Decode(&pkg)).To(Succeed())
				Expect(file.Close()).To(Succeed())

				pkg["engines"] = map[string]string{"npm": "^8"}
				content, err := json.Marshal(pkg)
				Expect(err).NotTo(HaveOccurred())
				Expect(os.WriteFile(filepath.Join(source, "package.json"), content, 0600)).To(Succeed())
			})

			it.After(func() {
				Expect(os.RemoveAll(source)).To(Succeed())
			})

			it("uses the specified npm version", func() {
				deployment, logs, err := platform.Deploy.
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				Eventually(deployment).Should(Serve("Hello, World!"))

				Expect(logs.String()).To(SatisfyAll(
					ContainSubstring("engines.npm (package.json): ^8"),
					ContainSubstring("Downloading and installing npm ^8"),
				))
			})
		})

		context("when there are unmet dependencies", func() {
			var source string

			it.Before(func() {
				var err error
				source, err = switchblade.Source(filepath.Join(fixtures, "npm"))
				Expect(err).NotTo(HaveOccurred())

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

			it.After(func() {
				Expect(os.RemoveAll(source)).To(Succeed())
			})

			it("prints a warning", func() {
				_, logs, err := platform.Deploy.
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				Expect(logs).To(ContainLines(
					ContainSubstring("Unmet dependencies don't fail npm install but may cause runtime issues"),
				))
			})
		})
	}
}
