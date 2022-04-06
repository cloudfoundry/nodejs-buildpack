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
		serviceName = "newrelic-" + cutlass.RandStringRunes(10)
		serviceOffering = "newrelic-" + cutlass.RandStringRunes(10)
		app = cutlass.New(Fixtures("with_newrelic"))
	})

	AfterEach(func() {
		app = DestroyApp(app)

		RunCF("purge-service-offering", "-f", serviceOffering)
		RunCF("delete-service-broker", "-f", serviceOffering)
		RunCF("delete-service", "-f", serviceName)

		serviceBrokerApp = DestroyApp(serviceBrokerApp)
	})

	Context("deploying a NodeJS app with NewRelic", func() {
		It("push a newrelic app without a service", func() {
			PushAppAndConfirm(app)
			Eventually(app.Stdout.String).Should(ContainSubstring("&license_key=fake_new_relic_key1"))
			Expect(app.Stdout.String()).ToNot(ContainSubstring("&license_key=fake_new_relic_key2"))
			Expect(app.Stdout.String()).ToNot(ContainSubstring("&license_key=fake_new_relic_key3"))
		})

		Context("with user provided service", func() {
			It("push a newrelic app with a user provided service", func() {
				Expect(RunCF("create-user-provided-service", serviceName, "-p", `{"licenseKey": "fake_new_relic_key3"}`)).To(Succeed())
				Expect(app.PushNoStart()).To(Succeed())
				Expect(RunCF("bind-service", app.Name, serviceName)).To(Succeed())
				PushAppAndConfirm(app)

				Eventually(app.Stdout.String).Should(ContainSubstring("&license_key=fake_new_relic_key3"))
				Expect(app.Stdout.String()).ToNot(ContainSubstring("&license_key=fake_new_relic_key1"))
				Expect(app.Stdout.String()).ToNot(ContainSubstring("&license_key=fake_new_relic_key2"))

				Expect(RunCF("unbind-service", app.Name, serviceName)).To(Succeed())
			})
		})

		Context("with a service broker", func() {
			BeforeEach(func() {
				serviceBrokerApp = cutlass.New(Fixtures("fake_newrelic_service_broker"))
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

			It("push a newrelic app with a marketplace provided service", func() {
				serviceFromBroker := "newrelic-sb-" + cutlass.RandStringRunes(10)
				RunCF("create-service-broker", serviceBrokerApp.Name, "username", "password", serviceBrokerURL, "--space-scoped")
				RunCF("create-service", serviceOffering, "public", serviceFromBroker)

				Expect(app.PushNoStart()).To(Succeed())
				Expect(RunCF("bind-service", app.Name, serviceFromBroker)).To(Succeed())
				Expect(app.Restart()).To(Succeed())

				Eventually(app.Stdout.String).Should(ContainSubstring("&license_key=fake_new_relic_key2"))
				Expect(app.Stdout.String()).ToNot(ContainSubstring("&license_key=fake_new_relic_key1"))
				Expect(app.Stdout.String()).ToNot(ContainSubstring("&license_key=fake_new_relic_key3"))
			})
		})
	})
})
