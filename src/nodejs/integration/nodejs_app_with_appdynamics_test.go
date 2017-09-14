package integration_test

import (
	"path/filepath"
	"time"

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

	PContext("deploying a NodeJS app with AppDynamics", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "with_appdynamics"))
		})

		It("tries to talk to AppDynamics with host-name from the env vars", func() {
			PushAppAndConfirm(app)
			Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
			Eventually(func() string { return app.Stdout.String() }, 5*time.Second).Should(ContainSubstring("appdynamics v"))
			Eventually(func() string { return app.Stdout.String() }, 5*time.Second).Should(ContainSubstring("starting control socket"))
			Eventually(func() string { return app.Stdout.String() }, 5*time.Second).Should(ContainSubstring("controllerHost: 'test-host'"))
		})
	})
})
