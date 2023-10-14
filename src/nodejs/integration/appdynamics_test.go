package integration_test

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/switchblade"
	"github.com/sclevine/spec"

	. "github.com/cloudfoundry/switchblade/matchers"
	. "github.com/onsi/gomega"
)

func testAppdynamics(platform switchblade.Platform, fixtures string) func(*testing.T, spec.G, spec.S) {
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

		for _, n := range []string{"appdynamics", "app-dynamics"} {
			service := "some-" + n

			context(fmt.Sprintf("with a service called %s", name), func() {
				it("ensures the service can be bound to the app", func() {
					deployment, _, err := platform.Deploy.
						WithServices(map[string]switchblade.Service{
							service: {
								"account-access-key": "test-key",
								"account-name":       "test-account",
								"host-name":          "test-ups-host",
								"port":               "1234",
								"ssl-enabled":        "true",
							},
						}).
						Execute(name, filepath.Join(fixtures, "services", "appdynamics"))
					Expect(err).NotTo(HaveOccurred())

					Eventually(deployment).Should(Serve("Hello, World!"))

					response, err := http.Get(fmt.Sprintf("%s/logs", deployment.ExternalURL))
					Expect(err).NotTo(HaveOccurred())
					defer response.Body.Close()

					logs, err := io.ReadAll(response.Body)
					Expect(err).NotTo(HaveOccurred())

					Expect(string(logs)).To(ContainSubstring("Starting AppDynamics Agent"))
					Expect(string(logs)).To(ContainSubstring("controller=test-account@test-ups-host:1234"))
					Expect(string(logs)).To(ContainSubstring("Application name: " + name))
				})
			})
		}

		context("when APPDYNAMICS_AGENT_APPLICATION_NAME is set", func() {
			it("uses that value", func() {
				deployment, _, err := platform.Deploy.
					WithServices(map[string]switchblade.Service{
						"appdynamics": {
							"account-access-key": "test-key",
							"account-name":       "test-account",
							"host-name":          "test-ups-host",
							"port":               "1234",
							"ssl-enabled":        "true",
						},
					}).
					WithEnv(map[string]string{
						"APPDYNAMICS_AGENT_APPLICATION_NAME": "set-name",
					}).
					Execute(name, filepath.Join(fixtures, "services", "appdynamics"))
				Expect(err).NotTo(HaveOccurred())

				Eventually(deployment).Should(Serve("Hello, World!"))

				response, err := http.Get(fmt.Sprintf("%s/logs", deployment.ExternalURL))
				Expect(err).NotTo(HaveOccurred())
				defer response.Body.Close()

				logs, err := io.ReadAll(response.Body)
				Expect(err).NotTo(HaveOccurred())

				Expect(string(logs)).To(ContainSubstring("Starting AppDynamics Agent"))
				Expect(string(logs)).To(ContainSubstring("controller=test-account@test-ups-host:1234"))
				Expect(string(logs)).To(ContainSubstring("Application name: set-name"))

				Expect(deployment).To(Serve("set-name").WithEndpoint("/name"))
			})
		})

		// A user sets $APPD_AGENT to instruct that a separate "appdynamics" buildpack is in charge of setting up appd.
		// See PR by appd staff https://github.com/cloudfoundry/nodejs-buildpack/pull/214
		context("when APPD_AGENT is set", func() {
			it("this buildpack does not setup appdynamics", func() {
				deployment, _, err := platform.Deploy.
					WithEnv(map[string]string{
						"APPD_AGENT": "nodejs",
					}).
					Execute(name, filepath.Join(fixtures, "services", "appdynamics"))
				Expect(err).NotTo(HaveOccurred())
				Eventually(deployment).Should(Serve("Hello, World!"))

				response, err := http.Get(fmt.Sprintf("%s/logs", deployment.ExternalURL))
				Expect(err).NotTo(HaveOccurred())
				defer response.Body.Close()

				logs, err := io.ReadAll(response.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(logs)).NotTo(ContainSubstring("Starting AppDynamics Agent"))
			})
		})
	}
}
