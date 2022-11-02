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

func testDefault(platform switchblade.Platform, fixtures string) func(*testing.T, spec.G, spec.S) {
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

		it("deploys a simple app", func() {
			deployment, logs, err := platform.Deploy.
				Execute(name, filepath.Join(fixtures, "simple"))
			Expect(err).NotTo(HaveOccurred())

			Expect(logs).To(ContainLines(MatchRegexp("Installing node 18\\.\\d+\\.\\d+")))

			Eventually(deployment).Should(Serve("Hello world!"))

			response, err := http.Get(fmt.Sprintf("%s/process", deployment.ExternalURL))
			Expect(err).NotTo(HaveOccurred())
			defer response.Body.Close()

			var process struct {
				Version string `json:"version"`
				Env     struct {
					NodeOptions     string `json:"NODE_OPTIONS"`
					NodeHome        string `json:"NODE_HOME"`
					NodeEnv         string `json:"NODE_ENV"`
					MemoryAvailable string `json:"MEMORY_AVAILABLE"`
				} `json:"env"`
			}
			Expect(json.NewDecoder(response.Body).Decode(&process)).To(Succeed())

			Expect(process.Version).To(MatchRegexp(`v18\.\d+\.\d+`))
			Expect(process.Env.NodeOptions).To(BeEmpty())
			Expect(process.Env.NodeHome).To(Equal("/home/vcap/deps/0/node"))
			Expect(process.Env.NodeEnv).To(Equal("production"))
			Expect(process.Env.MemoryAvailable).To(Equal("1024"))
		})
	}
}
