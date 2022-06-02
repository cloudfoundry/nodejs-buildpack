package integration_test

import (
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

	Context("deploying a Node.js app with a specified npm version", func() {
		BeforeEach(func() {
			app = cutlass.New(Fixtures("simple_app_with_npm_version"))
		})

		It("installs npm", func() {
			PushAppAndConfirm(app)
			Expect(app.Stdout.String()).To(ContainSubstring("engines.npm (package.json): ^7"))
			Expect(app.Stdout.String()).To(ContainSubstring("Downloading and installing npm ^7"))
		})
	})
})
