package integration_test

import (
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CF NodeJS Buildpack", func() {
	var app *cutlass.App
	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
	})

	Context("deploying a NodeJS app with NewRelic", func() {
		Context("when New Relic environment variables are set", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "with_newrelic"))
			})

			It("tries to talk to NewRelic with the license key from the env vars", func() {
				PushAppAndConfirm(app)
				Expect(app.Stdout.String()).To(ContainSubstring("&license_key=fake_new_relic_key2"))
				Expect(app.Stdout.String()).ToNot(ContainSubstring("&license_key=fake_new_relic_key1"))
			})
		})

		Context("when newrelic.js sets license_key", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "with_newrelic_js"))
			})

			It("tries to talk to NewRelic with the license key from newrelic.js", func() {
				PushAppAndConfirm(app)
				Expect(app.Stdout.String()).ToNot(ContainSubstring("&license_key=fake_new_relic_key2"))
				Expect(app.Stdout.String()).To(ContainSubstring("&license_key=fake_new_relic_key1"))
			})
		})
	})
})
