package brats_test

import (
	"github.com/cloudfoundry/libbuildpack/bratshelper"
	"github.com/cloudfoundry/libbuildpack/cutlass"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Nodejs buildpack", func() {
	bratshelper.UnbuiltBuildpack("node", CopyBrats)
	bratshelper.DeployingAnAppWithAnUpdatedVersionOfTheSameBuildpack(CopyBrats)
	bratshelper.StagingWithBuildpackThatSetsEOL("node", CopyBrats)
	bratshelper.StagingWithADepThatIsNotTheLatest("node", CopyBrats)
	bratshelper.StagingWithCustomBuildpackWithCredentialsInDependencies(`node\-[\d\.]+\-linux\-x64\-[\da-f]+\.tgz`, CopyBrats)
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
			Expect(app.GetBody("/bson-ext")).To(ContainSubstring("Hello Bson-ext!"))
		})
		By("installs the correct version", func() {
			Expect(app.Stdout.String()).To(ContainSubstring("Installing node " + nodeVersion))
		})
	})
})
