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

		command := exec.Command("cf", "purge-service-offering", "-f", "snyk")
		command.Stdout = GinkgoWriter
		command.Stderr = GinkgoWriter
		_ = command.Run()

		command = exec.Command("cf", "delete-service-broker", "-f", "snyk")
		command.Stdout = GinkgoWriter
		command.Stderr = GinkgoWriter
		_ = command.Run()

		sbApp = DestroyApp(sbApp)
	})

	BeforeEach(func() {
		sbApp = cutlass.New(filepath.Join(bpDir, "fixtures", "fake_snyk_service_broker"))
		Expect(sbApp.Push()).To(Succeed())
		Eventually(func() ([]string, error) { return sbApp.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))

		var err error
		sbUrl, err = sbApp.GetUrl("")
		Expect(err).ToNot(HaveOccurred())

		RunCf("create-service-broker", "snyk", "username", "password", sbUrl, "--space-scoped")
		RunCf("create-service", "snyk", "public", "snyk")

		app = cutlass.New(filepath.Join(bpDir, "fixtures", "with_snyk"))
		app.SetEnv("BP_DEBUG", "true")
		PushAppAndConfirm(app)
	})

	Context("bind NodeJS app with snyk service", func() {
		BeforeEach(func() {
			RunCf("bind-service", app.Name, "snyk")

			app.Stdout.Reset()
			RunCf("restage", app.Name)
		})

		It("test if Snyk token was found", func() {
			Expect(app.ConfirmBuildpack(buildpackVersion)).To(Succeed())
			Expect(app.Stdout.String()).To(ContainSubstring("Snyk token was found"))
		})
	})
})
