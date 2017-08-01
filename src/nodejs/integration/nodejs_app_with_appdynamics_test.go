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

	Context("deploying a NodeJS app with AppDynamics", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "with_appdynamics"))
		})

		It("tries to talk to AppDynamics with host-name from the env vars", func() {
			PushAppAndConfirm(app)
			Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
			Expect(app.Stdout.String()).To(ContainSubstring("appdynamics v"))
			Expect(app.Stdout.String()).To(ContainSubstring("starting control socket"))
			Expect(app.Stdout.String()).To(ContainSubstring("controllerHost: 'test-host'"))
		})
	})
})
