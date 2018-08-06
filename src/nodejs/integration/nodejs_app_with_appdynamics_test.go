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
		app, serviceBrokerApp                                             *cutlass.App
		serviceBrokerURL, serviceNameOne, serviceNameTwo, serviceOffering string
	)

	appConfig := func() (string, error) {
		return app.GetBody("/config")
	}

	BeforeEach(func() {
		serviceNameOne = "appdynamics-" + cutlass.RandStringRunes(20)
		serviceNameTwo = "app-dynamics-" + cutlass.RandStringRunes(20)
		serviceOffering = "appdynamics-" + cutlass.RandStringRunes(20)
	})

	AfterEach(func() {
		app = DestroyApp(app)

		RunCF("purge-service-offering", "-f", serviceOffering)
		RunCF("delete-service-broker", "-f", serviceBrokerApp.Name)
		RunCF("delete-service", "-f", serviceNameOne)
		RunCF("delete-service", "-f", serviceNameTwo)

		serviceBrokerApp = DestroyApp(serviceBrokerApp)
	})

	It("deploying a NodeJS app with appdynamics", func() {
		By("set up a service broker", func() {
			serviceBrokerApp = cutlass.New(filepath.Join(bpDir, "fixtures", "fake_appdynamics_service_broker"))
			serviceBrokerApp.SetEnv("OFFERING_NAME", serviceOffering)
			Expect(serviceBrokerApp.Push()).To(Succeed())
			Eventually(func() ([]string, error) { return serviceBrokerApp.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))

			var err error
			serviceBrokerURL, err = serviceBrokerApp.GetUrl("")
			Expect(err).ToNot(HaveOccurred())
		})

		app = cutlass.New(filepath.Join(bpDir, "fixtures", "with_appdynamics"))
		app.Name = "nodejs-appdynamics-" + cutlass.RandStringRunes(10)
		app.Memory = "256M"
		app.Disk = "512M"

		By("Pushing an app with a user provided service", func() {
			Expect(RunCF("create-user-provided-service", serviceNameOne, "-p", `{"host-name":"test-ups-host","port":"1234","account-name":"test-account","ssl-enabled":"true","account-access-key":"test-key"}`)).To(Succeed())

			Expect(app.PushNoStart()).To(Succeed())
			Expect(RunCF("bind-service", app.Name, serviceNameOne)).To(Succeed())
			Expect(app.Restart()).To(Succeed())
			Eventually(func() ([]string, error) { return app.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))

			Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))

			Expect(app.Stdout.String()).To(ContainSubstring("Appdynamics agent logs"))
			Eventually(appConfig, 10*time.Second).Should(ContainSubstring(`"controllerHost": "test-ups-host"`))
		})

		By("Unbinding and deleting the CUPS appdynamics service", func() {
			Expect(RunCF("unbind-service", app.Name, serviceNameOne)).To(Succeed())
			Expect(RunCF("delete-service", "-f", serviceNameOne)).To(Succeed())
		})

		By("Pushing an app with a user provided service named app-dynamics", func() {
			Expect(RunCF("create-user-provided-service", serviceNameTwo, "-p", `{"host-name":"test-ups-2-host","port":"1234","account-name":"test-account","ssl-enabled":"true","account-access-key":"test-key"}`)).To(Succeed())

			app.Stdout.Reset()

			Expect(RunCF("bind-service", app.Name, serviceNameTwo)).To(Succeed())
			Expect(app.Restart()).To(Succeed())
			Eventually(func() ([]string, error) { return app.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))

			Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
			Expect(app.Stdout.String()).To(ContainSubstring("Appdynamics agent logs"))
			Eventually(appConfig, 10*time.Second).Should(ContainSubstring(`"controllerHost": "test-ups-2-host"`))
		})

		By("Unbinding and deleting the CUPS appdynamics service", func() {
			Expect(RunCF("unbind-service", app.Name, serviceNameTwo)).To(Succeed())
			Expect(RunCF("delete-service", "-f", serviceNameTwo)).To(Succeed())
		})

		By("Pushing an app with a marketplace pr ovided service", func() {
			serviceFromBroker := "appdynamics-sb-" + cutlass.RandStringRunes(10)
			Expect(RunCF("create-service-broker", serviceBrokerApp.Name, "username", "password", serviceBrokerURL, "--space-scoped")).To(Succeed())
			Expect(RunCF("create-service", serviceOffering, "public", serviceFromBroker)).To(Succeed())

			app.Stdout.Reset()

			Expect(RunCF("bind-service", app.Name, serviceFromBroker)).To(Succeed())
			Expect(app.Restart()).To(Succeed())
			Eventually(func() ([]string, error) { return app.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))

			Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
			Expect(app.Stdout.String()).To(ContainSubstring("Appdynamics agent logs"))
			Eventually(appConfig, 10*time.Second).Should(ContainSubstring(`"controllerHost": "test-sb-host"`))
		})
	})
})
