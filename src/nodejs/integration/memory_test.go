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

func testMemory(platform switchblade.Platform, fixtures string) func(*testing.T, spec.G, spec.S) {
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

		it("is running with autosized max_old_space_size", func() {
			deployment, _, err := platform.Deploy.
				WithEnv(map[string]string{"OPTIMIZE_MEMORY": "true"}).
				Execute(name, filepath.Join(fixtures, "simple"))
			Expect(err).NotTo(HaveOccurred())

			Eventually(deployment).Should(Serve("Hello world!"))

			response, err := http.Get(fmt.Sprintf("%s/process", deployment.ExternalURL))
			Expect(err).NotTo(HaveOccurred())
			defer response.Body.Close()

			var process struct {
				Env struct {
					MemoryAvailable string `json:"MEMORY_AVAILABLE"`
					NodeOptions     string `json:"NODE_OPTIONS"`
				} `json:"env"`
			}
			Expect(json.NewDecoder(response.Body).Decode(&process)).To(Succeed())

			Expect(process.Env.MemoryAvailable).To(Equal("1024"))
			Expect(process.Env.NodeOptions).To(Equal("--max_old_space_size=768"))
		})
	}
}
