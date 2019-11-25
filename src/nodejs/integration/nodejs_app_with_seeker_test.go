package integration_test

import (
	"time"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CF NodeJS Buildpack", func() {
	var (
		app              *cutlass.App
		serviceBrokerApp *cutlass.App
		serviceNameOne   string
	)

	BeforeEach(func() {
		serviceNameOne = "seeker-" + cutlass.RandStringRunes(20)
	})

	AfterEach(func() {
		app = DestroyApp(app)

		_ = RunCF("delete-service", "-f", serviceNameOne)

		serviceBrokerApp = DestroyApp(serviceBrokerApp)
	})

	appConfig := func() (string, error) {
		return app.GetBody("/config")
	}

	It("deploying a NodeJS app with seeker", func() {
		app = cutlass.New(Fixtures("with_seeker"))
		app.Name = "nodejs-seeker-" + cutlass.RandStringRunes(10)
		app.Memory = "256M"
		app.Disk = "512M"

		app.SetEnv("BP_DEBUG", "true")
		app.SetEnv("SEEKER_APP_ENTRY_POINT", "server.js")
		app.SetEnv("SEEKER_AGENT_DOWNLOAD_URL", "https://github.com/synopsys-sig/seeker-nodejs-buildpack-test/releases/download/1.1/synopsys-sig-seeker-1.1.0.zip")

		By("Pushing an app with a user provided service", func() {
			Expect(RunCF("create-user-provided-service", serviceNameOne, "-p", `{
				"seeker_server_url": "http://non-existing-domain.seeker-test.com"
			}`)).To(Succeed())

			Expect(app.PushNoStart()).To(Succeed())
			Expect(RunCF("bind-service", app.Name, serviceNameOne)).To(Succeed())
			Expect(app.Restart()).To(Succeed())
			Eventually(func() ([]string, error) { return app.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))

			// test that the app was deployed successfully
			Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
			// test the the mocked agent was injected properly
			Expect(app.Stdout.String()).To(ContainSubstring("Hello from Seeker"))
			// test that the credentials were passed successfully
			Eventually(appConfig, 10*time.Second).Should(ContainSubstring("SEEKER_SERVER_URL: http://non-existing-domain.seeker-test.com"))
		})

		By("Unbinding and deleting the CUPS seeker service", func() {
			Expect(RunCF("unbind-service", app.Name, serviceNameOne)).To(Succeed())
			Expect(RunCF("delete-service", "-f", serviceNameOne)).To(Succeed())
		})
	})
})
