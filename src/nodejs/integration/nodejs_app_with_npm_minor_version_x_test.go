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

	Context("deploying a Node.js app with a specified npm version ending with .x", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "npm_version_with_minor_x"))
		})

		It("should not attempt to download npm because it should match existing version", func() {
			PushAppAndConfirm(app)
			Expect(app.Stdout.String()).ToNot(ContainSubstring("Downloading and installing npm 6.4.x"))
		})
	})
})
