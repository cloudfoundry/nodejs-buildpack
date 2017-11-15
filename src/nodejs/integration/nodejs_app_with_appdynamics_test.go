package integration_test

import (
	"os/exec"
	"path/filepath"
	"time"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CF NodeJS Buildpack", func() {
	var app, sbApp *cutlass.App
	var sbUrl string
	RunCf := func(args ...string) error {
		command := exec.Command("cf", args...)
		command.Stdout = GinkgoWriter
		command.Stderr = GinkgoWriter
		return command.Run()
	}

	AfterEach(func() {
		app = DestroyApp(app)

		_ = RunCf("purge-service-offering", "-f", "appdynamics")
		_ = RunCf("delete-service-broker", "-f", "appdynamics")
		_ = RunCf("delete-service", "-f", "appdynamics")
		_ = RunCf("delete-service", "-f", "app-dynamics")

		sbApp = DestroyApp(sbApp)
	})

	appConfig := func() (string, error) {
		return app.GetBody("/config")
	}

	It("deploying a NodeJS app with appdynamics", func() {
		By("set up a service broker", func() {
			sbApp = cutlass.New(filepath.Join(bpDir, "fixtures", "fake_appdynamics_service_broker"))
			Expect(sbApp.Push()).To(Succeed())
			Eventually(func() ([]string, error) { return sbApp.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))

			var err error
			sbUrl, err = sbApp.GetUrl("")
			Expect(err).ToNot(HaveOccurred())
		})

		app = cutlass.New(filepath.Join(bpDir, "fixtures", "with_appdynamics"))
		app.Memory = "256M"
		app.Disk = "512M"

		By("Pushing an app with a user provided service named appdynamics", func() {
			Expect(RunCf("create-user-provided-service", "appdynamics", "-p", `{"host-name":"test-ups-host","port":"1234","account-name":"test-account","ssl-enabled":"true","account-access-key":"test-key"}`)).To(Succeed())
			PushAppAndConfirm(app)
			Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))

			Expect(app.Stdout.String()).To(ContainSubstring("Appdynamics agent logs"))
			Eventually(appConfig, 10*time.Second).Should(ContainSubstring(`"controllerHost": "test-ups-host"`))
		})

		By("Unbinding and deleting the CUPS appdynamics service", func() {
			Expect(RunCf("unbind-service", app.Name, "appdynamics")).To(Succeed())
			Expect(RunCf("delete-service", "-f", "appdynamics")).To(Succeed())
		})

		By("Pushing an app with a user provided service named app-dynamics", func() {
			Expect(RunCf("create-user-provided-service", "app-dynamics", "-p", `{"host-name":"test-ups-2-host","port":"1234","account-name":"test-account","ssl-enabled":"true","account-access-key":"test-key"}`)).To(Succeed())
			Expect(RunCf("bind-service", app.Name, "app-dynamics")).To(Succeed())

			Expect(RunCf("restart", app.Name)).To(Succeed())
			Eventually(func() ([]string, error) { return app.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))

			Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
			Eventually(appConfig, 10*time.Second).Should(ContainSubstring(`"controllerHost": "test-ups-2-host"`))
		})

		By("Unbinding and deleting the CUPS appdynamics service", func() {
			Expect(RunCf("unbind-service", app.Name, "app-dynamics")).To(Succeed())
			Expect(RunCf("delete-service", "-f", "app-dynamics")).To(Succeed())
		})

		By("Pushing an app with a marketplace provided service", func() {
			Expect(RunCf("create-service-broker", "appdynamics", "username", "password", sbUrl, "--space-scoped")).To(Succeed())
			Expect(RunCf("create-service", "appdynamics", "public", "appdynamics")).To(Succeed())

			app.Stdout.Reset()
			PushAppAndConfirm(app)
			Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))

			Expect(app.Stdout.String()).To(ContainSubstring("Appdynamics agent logs"))
			Eventually(appConfig, 10*time.Second).Should(ContainSubstring(`"controllerHost": "test-sb-host"`))
		})
	})
})
