package integration_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/switchblade"
	"github.com/sclevine/spec"

	. "github.com/cloudfoundry/switchblade/matchers"
	. "github.com/onsi/gomega"
)

func testSeeker(platform switchblade.Platform, fixtures string) func(*testing.T, spec.G, spec.S) {
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

		it("uses the Seeker service binding", func() {
			deployment, _, err := platform.Deploy.
				WithEnv(map[string]string{
					"BP_DEBUG":                  "true",
					"SEEKER_APP_ENTRY_POINT":    "server.js",
					"SEEKER_AGENT_DOWNLOAD_URL": "https://github.com/synopsys-sig/seeker-nodejs-buildpack-test/releases/download/1.1/synopsys-sig-seeker-1.1.0.zip",
				}).
				WithServices(map[string]switchblade.Service{
					"seeker-service": {
						"seeker_server_url": "http://non-existing-domain.seeker-test.com",
					},
				}).
				Execute(name, filepath.Join(fixtures, "simple"))
			Expect(err).NotTo(HaveOccurred())

			Eventually(deployment).Should(Serve(HavePrefix("require('/home/vcap/app/seeker/node_modules/@synopsys-sig/seeker');")).WithEndpoint("/fs/server.js"))

			response, err := http.Get(fmt.Sprintf("%s/process", deployment.ExternalURL))
			Expect(err).NotTo(HaveOccurred())
			defer response.Body.Close()

			var process struct {
				Env struct {
					SeekerServerURL string `json:"SEEKER_SERVER_URL"`
				}
			}
			err = json.NewDecoder(response.Body).Decode(&process)
			Expect(err).NotTo(HaveOccurred())

			Expect(process.Env.SeekerServerURL).To(Equal("http://non-existing-domain.seeker-test.com"))
		})
	}
}
