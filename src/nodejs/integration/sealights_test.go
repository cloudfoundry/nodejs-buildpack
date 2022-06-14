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

func testSealights(platform switchblade.Platform, fixtures string) func(*testing.T, spec.G, spec.S) {
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

		context("when there is a Procfile", func() {
			it.Before(func() {
				Expect(os.WriteFile(filepath.Join(source, "Procfile"), []byte("web: node server.js"), 0600)).To(Succeed())
			})

			it("modifies the Procfile start command", func() {
				deployment, _, err := platform.Deploy.
					WithEnv(map[string]string{"SL_BUILD_SESSION_ID": "bs1"}).
					WithServices(map[string]switchblade.Service{
						"sealights-service": {
							"token": "token1",
						},
					}).
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				Eventually(deployment).Should(Serve(
					ContainSubstring("web: ./node_modules/.bin/slnodejs run  --useinitialcolor true  --token token1 --buildsessionid bs1  server.js"),
				).WithEndpoint("/fs/Procfile"))
			})

			context("when there is also a package.json", func() {
				it.Before(func() {
					var pkg map[string]interface{}
					content, err := os.ReadFile(filepath.Join(source, "package.json"))
					Expect(err).NotTo(HaveOccurred())
					Expect(json.Unmarshal(content, &pkg)).To(Succeed())

					pkg["scripts"] = map[string]string{
						"start": "node server.js",
					}

					content, err = json.Marshal(pkg)
					Expect(err).NotTo(HaveOccurred())

					Expect(os.WriteFile(filepath.Join(source, "package.json"), content, 0600)).To(Succeed())
					Expect(os.WriteFile(filepath.Join(source, "Procfile"), []byte("web: npm start"), 0600)).To(Succeed())
				})

				it("modifies the package.json command and not the Procfile", func() {
					deployment, _, err := platform.Deploy.
						WithEnv(map[string]string{"SL_BUILD_SESSION_ID": "bs1"}).
						WithServices(map[string]switchblade.Service{
							"sealights-service": {
								"token": "token1",
							},
						}).
						Execute(name, source)
					Expect(err).NotTo(HaveOccurred())

					Eventually(deployment).Should(Serve(ContainSubstring("web: npm start")).WithEndpoint("/fs/Procfile"))
					Eventually(deployment).Should(Serve(
						ContainSubstring(`"start":"./node_modules/.bin/slnodejs run  --useinitialcolor true  --token token1 --buildsessionid bs1  server.js"`),
					).WithEndpoint("/fs/package.json"))
				})
			})
		})

		context("when there is a package.json", func() {
			it.Before(func() {
				var pkg map[string]interface{}
				content, err := os.ReadFile(filepath.Join(source, "package.json"))
				Expect(err).NotTo(HaveOccurred())
				Expect(json.Unmarshal(content, &pkg)).To(Succeed())

				pkg["scripts"] = map[string]string{
					"start": "node server.js",
				}

				content, err = json.Marshal(pkg)
				Expect(err).NotTo(HaveOccurred())

				Expect(os.WriteFile(filepath.Join(source, "package.json"), content, 0600)).To(Succeed())
			})

			it("modifies the package.json command", func() {
				deployment, _, err := platform.Deploy.
					WithEnv(map[string]string{"SL_BUILD_SESSION_ID": "bs1"}).
					WithServices(map[string]switchblade.Service{
						"sealights-service": {
							"token": "token1",
						},
					}).
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				Eventually(deployment).Should(Serve(
					ContainSubstring(`"start":"./node_modules/.bin/slnodejs run  --useinitialcolor true  --token token1 --buildsessionid bs1  server.js"`),
				).WithEndpoint("/fs/package.json"))
			})
		})
	}
}
