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
	AfterEach(func() {
		command := exec.Command("cf", "purge-service-offering", "-f", "newrelic")
		command.Stdout = GinkgoWriter
		command.Stderr = GinkgoWriter
		_ = command.Run()

		command = exec.Command("cf", "delete-service-broker", "-f", "newrelic")
		command.Stdout = GinkgoWriter
		command.Stderr = GinkgoWriter
		_ = command.Run()

		app = DestroyApp(app)
		sbApp = DestroyApp(sbApp)
	})

	Context("deploying a NodeJS app with NewRelic", func() {
		Context("when New Relic environment variables are set", func() {
			BeforeEach(func() {
				sbApp = cutlass.New(filepath.Join(bpDir, "fixtures", "fake_newrelic_service_broker"))
				Expect(sbApp.Push()).To(Succeed())
				Eventually(func() ([]string, error) { return sbApp.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))

				sbUrl, err := sbApp.GetUrl("")
				Expect(err).ToNot(HaveOccurred())
				command := exec.Command("cf", "create-service-broker", "newrelic", "username", "password", sbUrl, "--space-scoped")
				command.Stdout = GinkgoWriter
				command.Stderr = GinkgoWriter
				Expect(command.Run()).To(Succeed())

				command = exec.Command("cf", "create-service", "newrelic", "public", "newrelic")
				command.Stdout = GinkgoWriter
				command.Stderr = GinkgoWriter
				Expect(command.Run()).To(Succeed())

				app = cutlass.New(filepath.Join(bpDir, "fixtures", "with_newrelic"))
			})

			It("tries to talk to NewRelic with the license key from the env vars", func() {
				PushAppAndConfirm(app)
				Eventually(app.Stdout.String).Should(ContainSubstring("&license_key=fake_new_relic_key2"))
				Expect(app.Stdout.String()).ToNot(ContainSubstring("&license_key=fake_new_relic_key1"))
			})
		})

		Context("when newrelic.js sets license_key", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "with_newrelic_js"))
			})

			It("tries to talk to NewRelic with the license key from newrelic.js", func() {
				PushAppAndConfirm(app)
				Eventually(app.Stdout.String).Should(ContainSubstring("&license_key=fake_new_relic_key1"))
				Expect(app.Stdout.String()).ToNot(ContainSubstring("&license_key=fake_new_relic_key2"))
			})
		})
	})
})
