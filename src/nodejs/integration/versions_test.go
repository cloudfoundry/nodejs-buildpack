package integration_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/switchblade"
	"github.com/sclevine/spec"

	. "github.com/cloudfoundry/switchblade/matchers"
	. "github.com/onsi/gomega"
)

func testVersions(platform switchblade.Platform, fixtures string) func(*testing.T, spec.G, spec.S) {
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

			source, err = switchblade.Source(filepath.Join(fixtures, "simple"))
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(platform.Delete.Execute(name)).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		context("when there is a .nvmrc file", func() {
			it.Before(func() {
				Expect(os.WriteFile(filepath.Join(source, ".nvmrc"), []byte("18"), 0600)).To(Succeed())
			})

			it("uses the Node version specified in the .nvmrc file", func() {
				deployment, logs, err := platform.Deploy.Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				Expect(logs).To(ContainLines(
					MatchRegexp("Installing node 18\\.\\d+\\.\\d+"),
				))

				Eventually(deployment).Should(Serve("Hello world!"))

				response, err := http.Get(fmt.Sprintf("%s/process", deployment.ExternalURL))
				Expect(err).NotTo(HaveOccurred())
				defer response.Body.Close()

				var process struct {
					Version string `json:"version"`
				}
				Expect(json.NewDecoder(response.Body).Decode(&process)).To(Succeed())

				Expect(process.Version).To(MatchRegexp(`v18\.\d+\.\d+`))
			})
		})

		context("when the package.json file specifies a node version", func() {
			it.Before(func() {
				file, err := os.OpenFile(filepath.Join(source, "package.json"), os.O_RDWR, 0600)
				Expect(err).NotTo(HaveOccurred())

				var pkg map[string]interface{}
				Expect(json.NewDecoder(file).Decode(&pkg)).To(Succeed())
				Expect(file.Close()).To(Succeed())

				pkg["engines"] = map[string]string{"node": "~>18"}
				content, err := json.Marshal(pkg)
				Expect(err).NotTo(HaveOccurred())
				Expect(os.WriteFile(filepath.Join(source, "package.json"), content, 0600)).To(Succeed())
			})

			it("uses the Node version specified in the package.json file", func() {
				deployment, logs, err := platform.Deploy.Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				Expect(logs).To(ContainLines(
					MatchRegexp("Installing node 18\\.\\d+\\.\\d+"),
				))

				Eventually(deployment).Should(Serve("Hello world!"))

				response, err := http.Get(fmt.Sprintf("%s/process", deployment.ExternalURL))
				Expect(err).NotTo(HaveOccurred())
				defer response.Body.Close()

				var process struct {
					Version string `json:"version"`
				}
				Expect(json.NewDecoder(response.Body).Decode(&process)).To(Succeed())

				Expect(process.Version).To(MatchRegexp(`v18\.\d+\.\d+`))
			})

			context("when that version is unsupported", func() {
				it.Before(func() {
					file, err := os.OpenFile(filepath.Join(source, "package.json"), os.O_RDWR, 0600)
					Expect(err).NotTo(HaveOccurred())

					var pkg map[string]interface{}
					Expect(json.NewDecoder(file).Decode(&pkg)).To(Succeed())
					Expect(file.Close()).To(Succeed())

					pkg["engines"] = map[string]string{"node": "9000.0.0"}
					content, err := json.Marshal(pkg)
					Expect(err).NotTo(HaveOccurred())
					Expect(os.WriteFile(filepath.Join(source, "package.json"), content, 0600)).To(Succeed())
				})

				it("displays a nice error message and fails gracefully", func() {
					_, logs, err := platform.Deploy.Execute(name, source)
					Expect(err).To(HaveOccurred())

					Expect(logs).To(ContainLines(
						ContainSubstring("Unable to install node: no match found for 9000.0.0"),
					))
				})
			})
		})
	}
}
