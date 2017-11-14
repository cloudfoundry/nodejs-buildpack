package integration_test

import (
	"os"
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
	RunCf := func(args ...string) {
		command := exec.Command("cf", args...)
		command.Stdout = GinkgoWriter
		command.Stderr = GinkgoWriter
		Expect(command.Run()).To(Succeed())
	}

	AfterEach(func() {
		app = DestroyApp(app)

		command := exec.Command("cf", "purge-service-offering", "-f", "newrelic")
		command.Stdout = GinkgoWriter
		command.Stderr = GinkgoWriter
		_ = command.Run()

		command = exec.Command("cf", "delete-service-broker", "-f", "newrelic")
		command.Stdout = GinkgoWriter
		command.Stderr = GinkgoWriter
		_ = command.Run()

		sbApp = DestroyApp(sbApp)
	})

	It("deploying a NodeJS app with NewRelic", func() {
		By("set up a service broker", func() {
			sbApp = cutlass.New(filepath.Join(bpDir, "fixtures", "fake_newrelic_service_broker"))
			Expect(sbApp.Push()).To(Succeed())
			Eventually(func() ([]string, error) { return sbApp.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))

			var err error
			sbUrl, err = sbApp.GetUrl("")
			Expect(err).ToNot(HaveOccurred())
		})

		app = cutlass.New(filepath.Join(bpDir, "fixtures", "with_newrelic"))
		Expect(os.Rename(filepath.Join(app.Path, "manifest.yml"), filepath.Join(app.Path, "manifest.orig.yml"))).To(Succeed())

		By("Pushing a newrelic app without a service", func() {
			PushAppAndConfirm(app)

			Eventually(app.Stdout.String).Should(ContainSubstring("&license_key=fake_new_relic_key1"))
			Expect(app.Stdout.String()).ToNot(ContainSubstring("&license_key=fake_new_relic_key2"))
			Expect(app.Stdout.String()).ToNot(ContainSubstring("&license_key=fake_new_relic_key3"))
		})

		Expect(os.Rename(filepath.Join(app.Path, "manifest.orig.yml"), filepath.Join(app.Path, "manifest.yml"))).To(Succeed())

		By("Pushing an app with a user provided service", func() {
			RunCf("create-user-provided-service", "newrelic", "-p", `{"licenseKey": "fake_new_relic_key3"}`)

			app.Stdout.Reset()
			PushAppAndConfirm(app)

			Eventually(app.Stdout.String).Should(ContainSubstring("&license_key=fake_new_relic_key3"))
			Expect(app.Stdout.String()).ToNot(ContainSubstring("&license_key=fake_new_relic_key1"))
			Expect(app.Stdout.String()).ToNot(ContainSubstring("&license_key=fake_new_relic_key2"))
		})

		By("Unbinding and deleting the CUPS newrelic service", func() {
			RunCf("unbind-service", app.Name, "newrelic")
			RunCf("delete-service", "-f", "newrelic")
		})

		By("Pushing an app with a marketplace provided service", func() {
			RunCf("create-service-broker", "newrelic", "username", "password", sbUrl, "--space-scoped")
			RunCf("create-service", "newrelic", "public", "newrelic")

			app.Stdout.Reset()
			PushAppAndConfirm(app)

			Eventually(app.Stdout.String).Should(ContainSubstring("&license_key=fake_new_relic_key2"))
			Expect(app.Stdout.String()).ToNot(ContainSubstring("&license_key=fake_new_relic_key1"))
			Expect(app.Stdout.String()).ToNot(ContainSubstring("&license_key=fake_new_relic_key3"))
		})
	})
})
