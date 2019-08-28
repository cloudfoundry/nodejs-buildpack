package brats_test

import (
	"strings"

	"github.com/cloudfoundry/libbuildpack/bratshelper"
	"github.com/cloudfoundry/libbuildpack/cutlass"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Nodejs buildpack", func() {
	bratshelper.UnbuiltBuildpack("node", CopyBrats)
	bratshelper.DeployingAnAppWithAnUpdatedVersionOfTheSameBuildpack(CopyBrats)
	bratshelper.StagingWithBuildpackThatSetsEOL("node", CopyBrats)
	bratshelper.StagingWithCustomBuildpackWithCredentialsInDependencies(CopyBrats)
	bratshelper.DeployAppWithExecutableProfileScript("node", CopyBrats)
	bratshelper.DeployAnAppWithSensitiveEnvironmentVariables(CopyBrats)
	bratshelper.ForAllSupportedVersions("node", CopyBrats, func(nodeVersion string, app *cutlass.App) {
		PushApp(app)

		By("runs a simple webserver", func() {
			Expect(app.GetBody("/")).To(ContainSubstring("Hello World!"))
		})
		By("supports bcrypt", func() {
			Expect(app.GetBody("/bcrypt")).To(ContainSubstring("Hello Bcrypt!"))
		})
		By("supports bson-ext", func() {
			if strings.HasPrefix(nodeVersion, "12") {
				// TODO: Bson-ext doesn't work with NodeJS 12 yet. When it does work (this fails), this can be removed.
				Expect(app.GetBody("/bson-ext")).To(ContainSubstring("502 Bad Gateway: Registered endpoint failed to handle the request."))
			} else {
				Expect(app.GetBody("/bson-ext")).To(ContainSubstring("Hello Bson-ext!"))
			}
		})
		By("installs the correct version", func() {
			Expect(app.Stdout.String()).To(ContainSubstring("Installing node " + nodeVersion))
		})
	})
})
