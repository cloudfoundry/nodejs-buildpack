package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/switchblade"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testSnyk(platform switchblade.Platform, fixtures string) func(*testing.T, spec.G, spec.S) {
	return func(t *testing.T, context spec.G, it spec.S) {
		var (
			Expect = NewWithT(t).Expect

			name   string
			source string
		)

		it.Before(func() {
			var err error
			name, err = switchblade.RandomName()
			Expect(err).NotTo(HaveOccurred())

			source, err = switchblade.Source(filepath.Join(fixtures, "simple"))
			Expect(err).NotTo(HaveOccurred())

			var pkg map[string]interface{}
			content, err := os.ReadFile(filepath.Join(source, "package.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(json.Unmarshal(content, &pkg)).To(Succeed())

			pkg["dependencies"] = map[string]interface{}{
				"snyk": "latest",
			}
			content, err = json.Marshal(pkg)
			Expect(err).NotTo(HaveOccurred())

			Expect(os.WriteFile(filepath.Join(source, "package.json"), content, 0600)).To(Succeed())
		})

		it.After(func() {
			Expect(platform.Delete.Execute(name)).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		it("uses the Snyk service binding", func() {
			_, logs, err := platform.Deploy.
				WithEnv(map[string]string{
					"BP_DEBUG":                "true",
					"SNYK_SEVERITY_THRESHOLD": "low",
				}).
				WithServices(map[string]switchblade.Service{
					"snyk-service": {
						"apiToken": "snyk-secret-token",
					},
				}).
				Execute(name, source)
			Expect(err).To(HaveOccurred()) // NOTE: installing the agent intentionally fails because the token is not valid

			Expect(logs.String()).To(SatisfyAll(
				ContainSubstring("Snyk token was found"),
				ContainSubstring("Run Snyk test..."),
			))
		})
	}
}
