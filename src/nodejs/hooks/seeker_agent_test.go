package hooks_test

import (
	"bytes"
	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/nodejs-buildpack/src/nodejs/hooks"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os"
	"path/filepath"
)

var _ = Describe("seekerHook", func() {
	var (
		err      error
		buildDir string
		depsDir  string
		depsIdx  string
		logger   *libbuildpack.Logger
		stager   *libbuildpack.Stager
		buffer   *bytes.Buffer
		seeker   hooks.SeekerAfterCompileHook
	)

	BeforeEach(func() {
		buildDir, err = ioutil.TempDir("", "nodejs-buildpack.build.")
		Expect(err).NotTo(HaveOccurred())

		depsDir, err = ioutil.TempDir("", "nodejs-buildpack.deps.")
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
	Describe("AfterCompile - obtain agent by directly downloading the agent", func() {
		var (
			oldVcapApplication string
			oldVcapServices    string
			oldBpDebug         string
		)
		BeforeEach(func() {
			oldVcapApplication = os.Getenv("VCAP_APPLICATION")
			oldVcapServices = os.Getenv("VCAP_SERVICES")
			oldBpDebug = os.Getenv("BP_DEBUG")
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
		})
		Context("VCAP_SERVICES contains seeker service - as a regular service", func() {
			BeforeEach(func() {
				os.Setenv("VCAP_APPLICATION", `{"name":"pcf app"}`)
				os.Setenv("VCAP_SERVICES", `{
					"seeker-security-service": [
					 {
					   "name": "seeker_instance",
					   "instance_name": "seeker_instance",
					   "binding_name": null,
					   "credentials": {
						"seeker_server_url": "http://10.120.9.117:9911"
					   },
					   "syslog_drain_url": null,
					   "volume_mounts": [],
					   "label": null,
					   "provider": null,
					   "plan": "default-seeker-plan-new",
					   "tags": [
						 "security",
						 "agent",
						 "monitoring"
					   ]
					 }
					],
					"2": [{"name":"mysql"}]}
				`)
			})
		})

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
			It("Prepends 'require('./seeker/node_modules/@synopsys-sig/seeker);' to the server.js file", func() {
				entryPointPath := filepath.Join(buildDir, "server.js")
				Expect(entryPointPath).ToNot(BeAnExistingFile())
				const mockedCode = "some mock javascript code"
				Expect(ioutil.WriteFile(entryPointPath, []byte(mockedCode), 0755)).To(Succeed())
				seeker.PrependRequire(stager)
				contents, err := ioutil.ReadFile(entryPointPath)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(contents)).To(Equal(hooks.SeekerRequire + mockedCode))
			})
			It("does not prepend 'require('./seeker/node_modules/@synopsys-sig/seeker);' to the server.js file if it already exist", func() {
				entryPointPath := filepath.Join(buildDir, "server.js")
				Expect(entryPointPath).ToNot(BeAnExistingFile())
				const mockedCode = hooks.SeekerRequire + "some mock javascript code"
				Expect(ioutil.WriteFile(entryPointPath, []byte(mockedCode), 0755)).To(Succeed())
				contents, err := ioutil.ReadFile(entryPointPath)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(contents)).To(Equal(mockedCode))
			})
			It("Fails to prepend the require to the server.js file - when the file does not exist", func() {
				entryPointPath := filepath.Join(buildDir, "server.js")
				Expect(entryPointPath).ToNot(BeAnExistingFile())
				err = seeker.AfterCompile(stager)
				Expect(err).To(MatchError(ContainSubstring("no such file or directory")))
			})

		})
	})
})
