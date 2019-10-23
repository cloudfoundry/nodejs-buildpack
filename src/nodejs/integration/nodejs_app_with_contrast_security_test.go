package integration_test

import (
	"time"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CF NodeJS Buildpack", func() {
	var (
		app, serviceBrokerApp                          *cutlass.App
		serviceBrokerURL, serviceName, serviceOffering string
	)

	BeforeEach(func() {
		serviceName = "contrast-security-" + cutlass.RandStringRunes(10)
		serviceOffering = "contrast-security-" + cutlass.RandStringRunes(10)
	})

	AfterEach(func() {
		app = DestroyApp(app)

		RunCF("purge-service-offering", "-f", serviceOffering)
		RunCF("delete-service-broker", "-f", serviceOffering)
		RunCF("delete-service", "-f", serviceName)

		serviceBrokerApp = DestroyApp(serviceBrokerApp)
	})

	It("deploying a NodeJS app with Contrast Security", func() {
		By("set up a service broker", func() {
			serviceBrokerApp = cutlass.New(Fixtures("fake_contrast_security_service_broker"))
			serviceBrokerApp.Buildpacks = []string{
				"https://github.com/cloudfoundry/ruby-buildpack#master",
			}
			serviceBrokerApp.SetEnv("OFFERING_NAME", serviceOffering)
			Expect(serviceBrokerApp.Push()).To(Succeed())
			Eventually(func() ([]string, error) { return serviceBrokerApp.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))

			var err error
			serviceBrokerURL, err = serviceBrokerApp.GetUrl("")
			Expect(err).ToNot(HaveOccurred())
		})

		By("Pushing an app with a marketplace provided service", func() {
			app = cutlass.New(Fixtures("simple_app"))
			PushAppAndConfirm(app)

			app.SetEnv("BP_DEBUG", "true")

			serviceFromBroker := "contrast-security-service-broker-app-" + cutlass.RandStringRunes(10)
			RunCF("create-service-broker", serviceBrokerApp.Name, "username", "password", serviceBrokerURL, "--space-scoped")
			RunCF("create-service", serviceOffering, "public", serviceFromBroker)

			app.Stdout.Reset()
			RunCF("bind-service", app.Name, serviceFromBroker)
			Expect(app.Restart()).To(Succeed())
			Expect(app.Stdout.String()).ToNot(ContainSubstring("Contrast Security successfully wrote"))
		})
	})
})
