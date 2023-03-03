package hooks_test

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/nodejs-buildpack/src/nodejs/hooks"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("seekerHook", func() {
	var (
		err           error
		buildDir      string
		depsDir       string
		depsIdx       string
		logger        *libbuildpack.Logger
		stager        *libbuildpack.Stager
		buffer        *bytes.Buffer
		seeker        hooks.SeekerAfterCompileHook
		seekerRequire = hooks.SeekerRequire[:len(hooks.SeekerRequire)-1]
	)

	BeforeEach(func() {
		buildDir, err = os.MkdirTemp("", "nodejs-buildpack.build.")
		Expect(err).NotTo(HaveOccurred())

		depsDir, err = os.MkdirTemp("", "nodejs-buildpack.deps.")
		Expect(err).NotTo(HaveOccurred())

		depsIdx = "07"
		err = os.MkdirAll(filepath.Join(depsDir, depsIdx), 0755)

		buffer = new(bytes.Buffer)
		logger = libbuildpack.NewLogger(buffer)

		seeker = hooks.SeekerAfterCompileHook{
			Log: logger,
		}
	})

	JustBeforeEach(func() {
		args := []string{buildDir, "", depsDir, depsIdx}
		stager = libbuildpack.NewStager(args, logger, &libbuildpack.Manifest{})
	})

	AfterEach(func() {

		err = os.RemoveAll(buildDir)
		Expect(err).NotTo(HaveOccurred())

		err = os.RemoveAll(depsDir)
		Expect(err).NotTo(HaveOccurred())
		os.RemoveAll(filepath.Join(os.TempDir(), "seeker_tmp"))
	})

	Describe("AfterCompile - adding agent require code to entry point", func() {
		var (
			oldVcapApplication string
			oldVcapServices    string
			oldBpDebug         string
		)
		BeforeEach(func() {
			oldVcapApplication = os.Getenv("VCAP_APPLICATION")
			oldVcapServices = os.Getenv("VCAP_SERVICES")
			oldBpDebug = os.Getenv("BP_DEBUG")
			os.Setenv("SEEKER_APP_ENTRY_POINT", "server.js")
		})
		AfterEach(func() {
			os.Setenv("VCAP_APPLICATION", oldVcapApplication)
			os.Setenv("VCAP_SERVICES", oldVcapServices)
			os.Setenv("BP_DEBUG", oldBpDebug)
		})

		Context("VCAP_SERVICES contains seeker service - as a user provided service", func() {
			BeforeEach(func() {
				os.Setenv("VCAP_APPLICATION", `{"name":"pcf app"}`)
				os.Setenv("VCAP_SERVICES", `{
				  "user-provided": [
					{
					  "name": "seeker_service_v2",
					  "instance_name": "seeker_service_v2",
					  "binding_name": null,
					  "credentials": {
						"seeker_server_url": "http://10.120.9.117:9911"
					  },
					  "syslog_drain_url": "",
					  "volume_mounts": [],
					  "label": "user-provided",
					  "tags": []
					}
				  ]
				 }`)
			})

			It("prepends "+seekerRequire+" to the server.js file", func() {
				entryPointPath := filepath.Join(buildDir, "server.js")
				Expect(entryPointPath).ToNot(BeAnExistingFile())

				const mockedCode = "some mock javascript code"
				Expect(os.WriteFile(entryPointPath, []byte(mockedCode), 0755)).To(Succeed())

				seeker.PrependRequire(stager)
				contents, err := os.ReadFile(entryPointPath)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(contents)).To(Equal(hooks.SeekerRequire + mockedCode))
			})

			It("does not prepend "+seekerRequire+" to the server.js file if require already exist", func() {
				entryPointPath := filepath.Join(buildDir, "server.js")
				Expect(entryPointPath).ToNot(BeAnExistingFile())

				const mockedCode = hooks.SeekerRequire + "some mock javascript code"
				Expect(os.WriteFile(entryPointPath, []byte(mockedCode), 0755)).To(Succeed())

				seeker.PrependRequire(stager)
				contents, err := os.ReadFile(entryPointPath)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(contents)).To(Equal(mockedCode))
			})

			It("fails to prepend the "+seekerRequire+" to the server.js file in case the file does not exist", func() {
				entryPointPath := filepath.Join(buildDir, "server.js")
				Expect(entryPointPath).ToNot(BeAnExistingFile())

				err = seeker.AfterCompile(stager)
				Expect(err).To(MatchError(ContainSubstring("no such file or directory")))
			})
		})
	})
})
