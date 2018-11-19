package hooks_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"bytes"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/golang/mock/gomock"

	"github.com/cloudfoundry/nodejs-buildpack/src/nodejs/hooks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=snyk.go --destination=mocks_snyk_test.go --package=hooks_test

var _ = Describe("snykHook", func() {
	var (
		err             error
		buildDir        string
		depsDir         string
		depsIdx         string
		logger          *libbuildpack.Logger
		stager          *libbuildpack.Stager
		mockCtrl        *gomock.Controller
		mockSnykCommand *MockSnykCommand
		buffer          *bytes.Buffer
		snyk            hooks.SnykHook
	)
	const snykAgentPath = "node_modules/snyk/cli"
	const snykAgentMain = "index.js"

	BeforeEach(func() {
		buildDir, err = ioutil.TempDir("", "nodejs-buildpack.build.")
		Expect(err).To(BeNil())
		depsDir, err = ioutil.TempDir("", "nodejs-buildpack.deps.")
		Expect(err).To(BeNil())

		err = os.MkdirAll(filepath.Join(buildDir, snykAgentPath), 0755)

		buffer = new(bytes.Buffer)
		logger = libbuildpack.NewLogger(buffer)

		mockCtrl = gomock.NewController(GinkgoT())
		mockSnykCommand = NewMockSnykCommand(mockCtrl)
		snyk = hooks.SnykHook{
			SnykCommand: mockSnykCommand,
			Log:         logger,
		}
	})

	JustBeforeEach(func() {
		args := []string{buildDir, "", depsDir, depsIdx}
		stager = libbuildpack.NewStager(args, logger, &libbuildpack.Manifest{})
	})

	AfterEach(func() {
		mockCtrl.Finish()

		err = os.RemoveAll(buildDir)
		Expect(err).To(BeNil())

		err = os.RemoveAll(depsDir)
		Expect(err).To(BeNil())
	})

	Describe("AfterCompile", func() {
		var (
			oldVcapApplication string
			oldVcapServices    string
			oldBpDebug         string
			oldSnykToken       string
		)
		BeforeEach(func() {
			oldVcapApplication = os.Getenv("VCAP_APPLICATION")
			oldVcapServices = os.Getenv("VCAP_SERVICES")
			oldBpDebug = os.Getenv("BP_DEBUG")
			oldSnykToken = os.Getenv("SNYK_TOKEN")

		})
		AfterEach(func() {
			os.Setenv("VCAP_APPLICATION", oldVcapApplication)
			os.Setenv("VCAP_SERVICES", oldVcapServices)
			os.Setenv("BP_DEBUG", oldBpDebug)
			os.Setenv("SNYK_TOKEN", oldSnykToken)
		})

		Context("Snyk Token is empty", func() {
			BeforeEach(func() {
				os.Setenv("VCAP_APPLICATION", `{"name":"JimBob"}`)
				os.Setenv("VCAP_SERVICES", "{}")
				os.Setenv("BP_DEBUG", "TRUE")
			})

			It("does nothing and succeeds", func() {
				err = snyk.AfterCompile(stager)
				Expect(err).To(BeNil())
				Expect(buffer.String()).To(ContainSubstring("Snyk token wasn't found"))
			})
		})

		Context("VCAP_SERVICES is empty", func() {
			BeforeEach(func() {
				os.Setenv("VCAP_APPLICATION", `{"name":"JimBob"}`)
				os.Setenv("VCAP_SERVICES", "{}")
				os.Setenv("BP_DEBUG", "TRUE")
			})

			It("does nothing and succeeds", func() {
				err = snyk.AfterCompile(stager)
				Expect(err).To(BeNil())
				Expect(buffer.String()).To(ContainSubstring("Snyk token wasn't found"))
			})
		})

		Context("Snyk Token is set as environment variable", func() {
			BeforeEach(func() {
				os.Setenv("VCAP_SERVICES", "{}")
				os.Setenv("BP_DEBUG", "TRUE")
				os.Setenv("SNYK_TOKEN", "MY_SECRET_TOKEN")
			})

			It("Snyk token was found", func() {
				mockSnykCommand.EXPECT().Output(buildDir, "node", filepath.Join(buildDir, snykAgentPath, snykAgentMain), "test", "-d")

				err = ioutil.WriteFile(filepath.Join(buildDir, snykAgentPath, snykAgentMain), []byte("snyk cli"), 0644)
				Expect(err).To(BeNil())
				err = snyk.AfterCompile(stager)
				Expect(err).To(BeNil())
				Expect(buffer.String()).To(ContainSubstring("Snyk token was found."))
			})

			It("Snyk agent exists", func() {
				mockSnykCommand.EXPECT().Output(buildDir, "node", filepath.Join(buildDir, snykAgentPath, snykAgentMain), "test", "-d")

				err = ioutil.WriteFile(filepath.Join(buildDir, snykAgentPath, snykAgentMain), []byte("snyk cli"), 0644)
				Expect(err).To(BeNil())

				err = snyk.AfterCompile(stager)
				Expect(err).To(BeNil())
				Expect(buffer.String()).To(ContainSubstring("Checking if Snyk agent exists..."))
				Expect(buffer.String()).To(ContainSubstring("Snyk agent exists"))
				Expect(buffer.String()).To(ContainSubstring("Snyk finished successfully"))
			})

			It("Snyk agent doesn't exist failed installation", func() {
				mockSnykCommand.EXPECT().Output(buildDir, "npm", "install", "-g", "snyk").Return("", errors.New("failed to install"))

				err = snyk.AfterCompile(stager)
				Expect(err).To(MatchError("failed to install"))
				Expect(buffer.String()).To(ContainSubstring("Checking if Snyk agent exists..."))
				Expect(buffer.String()).To(ContainSubstring("Snyk agent doesn't exist"))
				Expect(buffer.String()).To(ContainSubstring("Failed to install Snyk agent"))
			})

			It("Snyk agent doesn't exist successful installation", func() {
				gomock.InOrder(
					mockSnykCommand.EXPECT().Output(buildDir, "npm", "install", "-g", "snyk"),
					mockSnykCommand.EXPECT().Output(buildDir, filepath.Join(depsDir, "node", "bin", "snyk"), "test", "-d"),
				)
				err = snyk.AfterCompile(stager)
				Expect(err).To(BeNil())
				Expect(buffer.String()).To(ContainSubstring("Checking if Snyk agent exists..."))
				Expect(buffer.String()).To(ContainSubstring("Snyk agent doesn't exist"))
				Expect(buffer.String()).To(ContainSubstring("Snyk finished successfully"))
			})

			It("Snyk test no vulnerabilties found", func() {
				mockSnykCommand.EXPECT().Output(buildDir, "node", filepath.Join(buildDir, snykAgentPath, snykAgentMain), "test", "-d")

				err = ioutil.WriteFile(filepath.Join(buildDir, snykAgentPath, snykAgentMain), []byte("snyk cli"), 0644)
				Expect(err).To(BeNil())

				err = snyk.AfterCompile(stager)
				Expect(err).To(BeNil())
				Expect(buffer.String()).To(ContainSubstring("Checking if Snyk agent exists..."))
				Expect(buffer.String()).To(ContainSubstring("Run Snyk test"))
				Expect(buffer.String()).To(ContainSubstring("Snyk finished successfully"))
			})

			It("Snyk test find vulnerabilties and failed", func() {
				mockSnykCommand.EXPECT().Output(buildDir, "node", filepath.Join(buildDir, snykAgentPath, snykAgentMain), "test", "-d").Return("dependencies for known", errors.New("vulns found"))

				err = ioutil.WriteFile(filepath.Join(buildDir, snykAgentPath, snykAgentMain), []byte("snyk cli"), 0644)
				Expect(err).To(BeNil())

				err = snyk.AfterCompile(stager)
				Expect(err).To(MatchError("vulns found"))
				Expect(buffer.String()).To(ContainSubstring("Checking if Snyk agent exists..."))
				Expect(buffer.String()).To(ContainSubstring("Run Snyk test"))
				Expect(buffer.String()).To(ContainSubstring("Snyk found vulnerabilties"))
			})

			It("Snyk test find vulnerabilties and continue", func() {
				os.Setenv("SNYK_DONT_BREAK_BUILD", "true")
				mockSnykCommand.EXPECT().Output(buildDir, "node", filepath.Join(buildDir, snykAgentPath, snykAgentMain), "test", "-d").Return("dependencies for known", errors.New("vulns found"))

				err = ioutil.WriteFile(filepath.Join(buildDir, snykAgentPath, snykAgentMain), []byte("snyk cli"), 0644)
				Expect(err).To(BeNil())

				err = snyk.AfterCompile(stager)
				Expect(err).To(BeNil())
				Expect(buffer.String()).To(ContainSubstring("Checking if Snyk agent exists..."))
				Expect(buffer.String()).To(ContainSubstring("Run Snyk test"))
				Expect(buffer.String()).To(ContainSubstring("SNYK_DONT_BREAK_BUILD was defined"))
				Expect(buffer.String()).To(ContainSubstring("Snyk finished successfully"))
			})
		})

		Context("VCAP_SERVICES has non snyk services", func() {
			BeforeEach(func() {
				os.Setenv("SNYK_TOKEN", "")
				os.Setenv("BP_DEBUG", "TRUE")
				os.Setenv("VCAP_APPLICATION", `{"name":"JimBob"}`)
				os.Setenv("VCAP_SERVICES", `{
					"0": [{"name":"mysql"}],
					"1": [{"name":"redis"}]
				}`)
			})

			It("does nothing and succeeds", func() {
				err = snyk.AfterCompile(stager)
				Expect(err).To(BeNil())

				Expect(buffer.String()).To(ContainSubstring("Snyk token wasn't found"))
			})
		})

		Context("VCAP_SERVICES has snyk service", func() {
			BeforeEach(func() {
				os.Setenv("VCAP_SERVICES", `{
						"0": [{"name":"mysql"}],
						"snyk-broker": [{"name":"snyk","credentials":{"apiToken":"SECRET_TOKEN"}}],
						"2": [{"name":"redis"}]
					}`)
				os.Setenv("SNYK_TOKEN", "")
				os.Setenv("BP_DEBUG", "TRUE")
			})

			It("Snyk token was found", func() {
				mockSnykCommand.EXPECT().Output(buildDir, "node", filepath.Join(buildDir, snykAgentPath, snykAgentMain), "test", "-d")

				err = ioutil.WriteFile(filepath.Join(buildDir, snykAgentPath, snykAgentMain), []byte("snyk cli"), 0644)
				Expect(err).To(BeNil())

				err = snyk.AfterCompile(stager)
				Expect(os.Getenv("SNYK_TOKEN")).To(Equal("SECRET_TOKEN"))
				Expect(os.Getenv("SNYK_API")).To(Equal(""))
				Expect(os.Getenv("SNYK_ORG_NAME")).To(Equal(""))
				Expect(err).To(BeNil())
				Expect(buffer.String()).To(ContainSubstring("Snyk token was found."))
				Expect(buffer.String()).To(ContainSubstring("Snyk finished successfully"))
			})
		})

		Context("VCAP_SERVICES ignore if snyk service is not the key", func() {
			BeforeEach(func() {
				os.Setenv("VCAP_SERVICES", `{
						"0": [{"name":"mysql"}],
						"1": [{"name":"snyk","credentials":{"apiToken":"SECRET_TOKEN"}}],
						"2": [{"name":"redis"}]
					}`)
				os.Setenv("SNYK_TOKEN", "")
				os.Setenv("BP_DEBUG", "TRUE")
			})

			It("does nothing and succeeds", func() {
				err = snyk.AfterCompile(stager)
				Expect(err).To(BeNil())
				Expect(buffer.String()).To(ContainSubstring("Snyk token wasn't found"))
			})
		})

		Context("VCAP_SERVICES has snyk service and monitor enabled", func() {
			BeforeEach(func() {
				os.Setenv("VCAP_SERVICES", `{
						"0": [{"name":"mysql"}],
						"snyk-broker-external": [{"name":"snyk-service-broker-external","credentials":{"apiToken":"SECRET_TOKEN"}}],
						"2": [{"name":"redis"}]
					}`)
				os.Setenv("VCAP_APPLICATION", `{"name":"monitored_app"}`)
				os.Setenv("SNYK_MONITOR_BUILD", "True")
				os.Setenv("BP_DEBUG", "TRUE")
			})

			It("Snyk agent not exists install and test Snyk", func() {
				gomock.InOrder(
					mockSnykCommand.EXPECT().Output(buildDir, "npm", "install", "-g", "snyk"),
					mockSnykCommand.EXPECT().Output(buildDir, filepath.Join(depsDir, "node", "bin", "snyk"), "test", "-d"),
					mockSnykCommand.EXPECT().Output(buildDir, filepath.Join(depsDir, "node", "bin", "snyk"), "monitor", "--project-name=monitored_app", "-d"),
				)

				err = snyk.AfterCompile(stager)
				Expect(os.Getenv("SNYK_TOKEN")).To(Equal("SECRET_TOKEN"))
				Expect(err).To(BeNil())
				Expect(buffer.String()).To(ContainSubstring("Snyk token was found."))
				Expect(buffer.String()).To(ContainSubstring("Run Snyk monitor..."))
				Expect(buffer.String()).To(ContainSubstring("Snyk finished successfully"))
			})
		})
		Context("We set a severity threshold", func() {
			var currentSeverityThreshold string
			BeforeEach(func() {
				os.Setenv("VCAP_SERVICES", `{
						"0": [{"name":"mysql"}],
						"snyk-broker-external": [{"name":"snyk-service-broker-external","credentials":{"apiToken":"SECRET_TOKEN"}}],
						"2": [{"name":"redis"}]
					}`)
				os.Setenv("VCAP_APPLICATION", `{"name":"monitored_app"}`)
				os.Setenv("BP_DEBUG", "TRUE")
				currentSeverityThreshold = "low"
				os.Setenv("SNYK_SEVERITY_THRESHOLD", currentSeverityThreshold)
			})

			AfterEach(func() {
				os.Setenv("SNYK_SEVERITY_THRESHOLD", "")
			})

			It("should add severity threshold to command args", func() {
				gomock.InOrder(
					mockSnykCommand.EXPECT().Output(buildDir, "npm", "install", "-g", "snyk"),
					mockSnykCommand.EXPECT().Output(buildDir, filepath.Join(depsDir, "node", "bin", "snyk"), "test", "-d", fmt.Sprintf("--severity-threshold=%s", currentSeverityThreshold)),
					mockSnykCommand.EXPECT().Output(buildDir, filepath.Join(depsDir, "node", "bin", "snyk"), "monitor", "--project-name=monitored_app", "-d", fmt.Sprintf("--severity-threshold=%s", currentSeverityThreshold)),
				)
				err = snyk.AfterCompile(stager)
				Expect(err).To(BeNil())
				Expect(buffer.String()).To(ContainSubstring("SNYK_SEVERITY_THRESHOLD is set to: %s", currentSeverityThreshold))
			})
		})

		Context("VCAP_SERVICES has snyk service and monitor enabled", func() {
			BeforeEach(func() {
				os.Setenv("VCAP_SERVICES", `{
						"0": [{"name":"mysql"}],
						"snyk-broker-external": [{"name":"snyk-service-broker-external","credentials":{"apiToken":"SECRET_TOKEN", "orgName": "my-org-name", "apiUrl": "https://staging.snyk.io/api"}}],
						"2": [{"name":"redis"}]
					}`)
				os.Setenv("VCAP_APPLICATION", `{"name":"monitored_app"}`)
				os.Setenv("SNYK_MONITOR_BUILD", "True")
			})

			It("Snyk agent not exists install and test Snyk", func() {
				gomock.InOrder(
					mockSnykCommand.EXPECT().Output(buildDir, "npm", "install", "-g", "snyk"),
					mockSnykCommand.EXPECT().Output(buildDir, filepath.Join(depsDir, "node", "bin", "snyk"), "test", "--org=my-org-name"),
					mockSnykCommand.EXPECT().Output(buildDir, filepath.Join(depsDir, "node", "bin", "snyk"), "monitor", "--project-name=monitored_app", "--org=my-org-name"),
				)

				err = snyk.AfterCompile(stager)
				Expect(os.Getenv("SNYK_TOKEN")).To(Equal("SECRET_TOKEN"))
				Expect(os.Getenv("SNYK_API")).To(Equal("https://staging.snyk.io/api"))
				Expect(os.Getenv("SNYK_ORG_NAME")).To(Equal("my-org-name"))
				Expect(err).To(BeNil())
				Expect(buffer.String()).To(ContainSubstring("Snyk finished successfully"))
			})
		})
	})
})
