package integration_test

import (
	"os/exec"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CF NodeJS Buildpack", func() {
	var app *cutlass.App
	var serviceName string
	RunCf := func(args ...string) error {
		command := exec.Command("cf", args...)
		command.Stdout = GinkgoWriter
		command.Stderr = GinkgoWriter
		return command.Run()
	}
	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil

		Expect(RunCf("delete-service", "-f", serviceName)).To(Succeed())
	})

	BeforeEach(func() {
		app = cutlass.New(filepath.Join(bpDir, "fixtures", "logenv"))
		app.SetEnv("BP_DEBUG", "true")

		PushAppAndConfirm(app)

		serviceName = "snyk-" + cutlass.RandStringRunes(20) + "-service"
	})

	Context("deploying a NodeJS app with Snyk service", func() {
		BeforeEach(func() {
			Expect(RunCf("cups", serviceName, "-p", "'{\"SNYK_TOKEN\":\"secrettoken\"}'")).To(Succeed())
			Expect(RunCf("bind-service", app.Name, serviceName)).To(Succeed())

			_ = RunCf("restage", app.Name)
		})

		It("test if Snyk token was found", func() {
			Expect(app.ConfirmBuildpack(buildpackVersion)).To(Succeed())
			Expect(app.Stdout.String()).To(ContainSubstring("Snyk token was found."))
		})
	})
})
