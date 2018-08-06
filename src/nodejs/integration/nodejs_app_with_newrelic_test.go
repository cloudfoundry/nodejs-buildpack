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
		app, serviceBrokerApp         *cutlass.App
		serviceBrokerURL, serviceName, serviceOffering string
	)

	BeforeEach(func() {
		serviceName = "newrelic-" + cutlass.RandStringRunes(10)
		serviceOffering = "newrelic-" + cutlass.RandStringRunes(10)
	})

	AfterEach(func() {
		app = DestroyApp(app)

		RunCF("purge-service-offering", "-f", serviceOffering)
		RunCF("delete-service-broker", "-f", serviceOffering)
		RunCF("delete-service", "-f", serviceName)

		serviceBrokerApp = DestroyApp(serviceBrokerApp)
	})

	It("deploying a NodeJS app with NewRelic", func() {
		By("set up a service broker", func() {
			serviceBrokerApp = cutlass.New(filepath.Join(bpDir, "fixtures", "fake_newrelic_service_broker"))
			serviceBrokerApp.SetEnv("OFFERING_NAME", serviceOffering)
			Expect(serviceBrokerApp.Push()).To(Succeed())
			Eventually(func() ([]string, error) { return serviceBrokerApp.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))

			var err error
			serviceBrokerURL, err = serviceBrokerApp.GetUrl("")
			Expect(err).ToNot(HaveOccurred())
		})

		app = cutlass.New(filepath.Join(bpDir, "fixtures", "with_newrelic"))

		By("Pushing a newrelic app without a service", func() {
			PushAppAndConfirm(app)

			Eventually(app.Stdout.String).Should(ContainSubstring("&license_key=fake_new_relic_key1"))
			Expect(app.Stdout.String()).ToNot(ContainSubstring("&license_key=fake_new_relic_key2"))
			Expect(app.Stdout.String()).ToNot(ContainSubstring("&license_key=fake_new_relic_key3"))
		})

		By("Pushing an app with a user provided service", func() {
			RunCF("create-user-provided-service", serviceName, "-p", `{"licenseKey": "fake_new_relic_key3"}`)

			app.Stdout.Reset()
			RunCF("bind-service", app.Name, serviceName)
			Expect(app.Restart()).To(Succeed())

			Eventually(app.Stdout.String).Should(ContainSubstring("&license_key=fake_new_relic_key3"))
			Expect(app.Stdout.String()).ToNot(ContainSubstring("&license_key=fake_new_relic_key1"))
			Expect(app.Stdout.String()).ToNot(ContainSubstring("&license_key=fake_new_relic_key2"))
		})

		By("Unbinding and deleting the CUPS newrelic service", func() {
			RunCF("unbind-service", app.Name, serviceName)
			RunCF("delete-service", "-f", serviceName)
		})

		By("Pushing an app with a marketplace provided service", func() {
			serviceFromBroker := "newrelic-sb-" + cutlass.RandStringRunes(10)
			RunCF("create-service-broker", serviceBrokerApp.Name, "username", "password", serviceBrokerURL, "--space-scoped")
			RunCF("create-service", serviceOffering, "public", serviceFromBroker)

			app.Stdout.Reset()
			RunCF("bind-service", app.Name, serviceFromBroker)
			Expect(app.Restart()).To(Succeed())

			Eventually(app.Stdout.String).Should(ContainSubstring("&license_key=fake_new_relic_key2"))
			Expect(app.Stdout.String()).ToNot(ContainSubstring("&license_key=fake_new_relic_key1"))
			Expect(app.Stdout.String()).ToNot(ContainSubstring("&license_key=fake_new_relic_key3"))
		})
	})
})
