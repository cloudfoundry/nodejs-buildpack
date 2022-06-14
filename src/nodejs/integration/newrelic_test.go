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

func testNewRelic(platform switchblade.Platform, fixtures string) func(*testing.T, spec.G, spec.S) {
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

		it("pushes a newrelic app with a user provided service", func() {
			deployment, _, err := platform.Deploy.
				WithEnv(map[string]string{"BP_DEBUG": "true"}).
				WithServices(map[string]switchblade.Service{
					"newrelic-service": {
						"licenseKey": "some-newrelic-key",
					},
				}).
				Execute(name, filepath.Join(fixtures, "services", "newrelic"))
			Expect(err).NotTo(HaveOccurred())

			Eventually(deployment).Should(Serve("Hello, World!"))

			response, err := http.Get(fmt.Sprintf("%s/process", deployment.ExternalURL))
			Expect(err).NotTo(HaveOccurred())
			defer response.Body.Close()

			var process struct {
				Env struct {
					NewRelicLicenseKey string `json:"NEW_RELIC_LICENSE_KEY"`
				} `json:"env"`
			}
			err = json.NewDecoder(response.Body).Decode(&process)
			Expect(err).NotTo(HaveOccurred())

			Expect(process.Env.NewRelicLicenseKey).To(Equal("some-newrelic-key"))
		})
	}
}
