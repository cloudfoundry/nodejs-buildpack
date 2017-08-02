package integration_test

import (
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Node.js applications with unmet dependencies", func() {
	var app *cutlass.App
	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
	})

	Context("package manager is npm", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "unmet_dep_npm"))
		})

		It("warns that unmet dependencies may cause issues", func() {
			PushAppAndConfirm(app)
			Expect(app.Stdout.String()).To(ContainSubstring("Unmet dependencies don't fail npm install but may cause runtime issues"))
		})
	})

	Context("package manager is yarn", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "unmet_dep_yarn"))
		})

		It("warns that unmet dependencies may cause issues", func() {
			PushAppAndConfirm(app)
			Expect(app.Stdout.String()).To(ContainSubstring("Unmet dependencies don't fail yarn install but may cause runtime issues"))
		})
	})
})
