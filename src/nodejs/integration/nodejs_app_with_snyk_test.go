package integration_test

import (
	"path/filepath"
	"time"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CF NodeJS Buildpack", func() {

	var (
		app, serviceBrokerApp *cutlass.App
		serviceBrokerURL      string
		serviceOffering = "snyk" + cutlass.RandStringRunes(10)
	)

	AfterEach(func() {
		app = DestroyApp(app)

		RunCF("purge-service-offering", "-f", serviceOffering)
		RunCF("delete-service-broker", "-f", serviceOffering)

		serviceBrokerApp = DestroyApp(serviceBrokerApp)
	})

	It("bind NodeJS app with snyk service", func() {
		By("set up a service broker", func() {
			serviceBrokerApp = cutlass.New(filepath.Join(bpDir, "fixtures", "fake_snyk_service_broker"))
			serviceBrokerApp.SetEnv("OFFERING_NAME", serviceOffering)
			Expect(serviceBrokerApp.Push()).To(Succeed())
			Eventually(func() ([]string, error) { return serviceBrokerApp.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))

			var err error
			serviceBrokerURL, err = serviceBrokerApp.GetUrl("")
			Expect(err).ToNot(HaveOccurred())
		})

		By("Pushing an app with a marketplace provided service", func() {
			serviceFromBroker := "snyk-sb-" + cutlass.RandStringRunes(10)
			RunCF("create-service-broker", serviceBrokerApp.Name, "username", "password", serviceBrokerURL, "--space-scoped")
			RunCF("create-service", serviceOffering, "public", serviceFromBroker)

			app = cutlass.New(filepath.Join(bpDir, "fixtures", "with_snyk"))
			app.SetEnv("BP_DEBUG", "true")
			Expect(app.PushNoStart()).To(Succeed())

			app.Stdout.Reset()
			RunCF("bind-service", app.Name, serviceFromBroker)
			app.Restart()

			Eventually(app.Stdout.String()).Should(ContainSubstring("Snyk token was found"))
			Eventually(app.Stdout.String()).ShouldNot(ContainSubstring("Missing node_modules folder: we can't test without dependencies"))
		})
	})
})
