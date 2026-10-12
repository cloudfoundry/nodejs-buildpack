package hooks_test

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/nodejs-buildpack/src/nodejs/hooks"

	"github.com/cloudfoundry/libbuildpack"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("contrastSecurityHook", func() {
	var (
		buffer   *bytes.Buffer
		logger   *libbuildpack.Logger
		contrast hooks.ContrastSecurityHook
		stager   *libbuildpack.Stager
	)

	BeforeEach(func() {
		buffer = new(bytes.Buffer)
		logger = libbuildpack.NewLogger(buffer)

		contrast = hooks.ContrastSecurityHook{
			Log: logger,
		}
	})

	Describe("AfterCompile", func() {

		JustBeforeEach(func() {
			tmpDir, _ := os.MkdirTemp("", "contrast_security_test")
			args := []string{tmpDir, "", ".", ""}
			stager = libbuildpack.NewStager(args, logger, &libbuildpack.Manifest{})
		})

		AfterEach(func() {
			Expect(os.RemoveAll(stager.BuildDir())).To(Succeed())
			Expect(os.RemoveAll(filepath.Join(stager.DepDir(), "profile.d"))).To(Succeed())
		})

		Context("Contrast Security credentials in VCAP_SERVICES", func() {
			BeforeEach(func() {
				os.Setenv("VCAP_APPLICATION", `{}`)
				os.Setenv("VCAP_SERVICES", `{
                                                "contrast-security": [
                                                 {
                                                  "binding_name": "CCC",
                                                  "credentials": {
                                                   "api_key": "sample_api_key",
                                                   "org_uuid": "sampe_org_uuid",
                                                   "service_key": "sample_service_key",
                                                   "teamserver_url": "sample_teamserver_url",
                                                   "username": "username@example.com"
                                                  }
                                                 }
                                                ]
                                               }`)
			})

			It("writes the Contrast Security credentials to a file in profile.d/", func() {
				err := contrast.AfterCompile(stager)
				Expect(err).To(BeNil())

				profileDir := filepath.Join(stager.DepDir(), "profile.d")
				files, err := os.ReadDir(profileDir)
				Expect(err).To(BeNil())
				//Expect(len(files)).To(Equal(1))

				var fileExists bool
				var fileIndex int
				for index, file := range files {
					if strings.Contains(file.Name(), "contrast_security") {
						fmt.Println(file.Name())
						fileExists = true
						fileIndex = index
					}
				}
				Expect(fileExists).To(Equal(true))

				fileBytes, err := os.ReadFile(filepath.Join(profileDir, files[fileIndex].Name()))

				if err != nil {
					Fail(err.Error())
				}

				fileContents := string(fileBytes)
				var sampleExportList string = "export CONTRAST__API__API_KEY='sample_api_key'\n" +
					"export CONTRAST__API__URL='sample_teamserver_url/Contrast'\n" +
					"export CONTRAST__API__SERVICE_KEY='sample_service_key'\n" +
					"export CONTRAST__API__USER_NAME='username@example.com'\n"
				Expect(fileContents).To(Equal(sampleExportList))
			})

		})

		Context("Contrast Security credentials in user-provided VCAP_SERVICES", func() {
			BeforeEach(func() {
				os.Setenv("VCAP_APPLICATION", `{}`)
				os.Setenv("VCAP_SERVICES", `{"user-provided":[
														{ "label": "user-provided",
															"name": "contrast-security-service",
															"tags": [ ],
															"instance_name": "sample_instance_name",
															"binding_name": null,
															"credentials": {
															"api_key": "sample_api_key",
															"service_key": "sample_service_key",
															"teamserver_url": "sample_teamserver_url",
															"username": "username@example.com"
															},
															"syslog_drain_url": "",
															"volume_mounts": [ ]
															}
													    ]}`)
			})

			It("writes the Contrast Security credentials to a file in profile.d/", func() {
				err := contrast.AfterCompile(stager)
				Expect(err).To(BeNil())

				profileDir := filepath.Join(stager.DepDir(), "profile.d")
				files, err := os.ReadDir(profileDir)
				Expect(err).To(BeNil())
				//Expect(len(files)).To(Equal(1))

				var fileExists bool
				var fileIndex int
				for index, file := range files {
					if strings.Contains(file.Name(), "contrast_security") {
						fmt.Println(file.Name())
						fileExists = true
						fileIndex = index
					}
				}
				Expect(fileExists).To(Equal(true))

				fileBytes, err := os.ReadFile(filepath.Join(profileDir, files[fileIndex].Name()))

				if err != nil {
					Fail(err.Error())
				}

				fileContents := string(fileBytes)
				var sampleExportList string = "export CONTRAST__API__API_KEY='sample_api_key'\n" +
					"export CONTRAST__API__URL='sample_teamserver_url/Contrast'\n" +
					"export CONTRAST__API__SERVICE_KEY='sample_service_key'\n" +
					"export CONTRAST__API__USER_NAME='username@example.com'\n"
				Expect(fileContents).To(Equal(sampleExportList))
			})
		})

		Context("teamserver_url already contains the /Contrast suffix", func() {
			BeforeEach(func() {
				os.Setenv("VCAP_APPLICATION", `{}`)
				os.Setenv("VCAP_SERVICES", `{
					"contrast-security": [{
						"credentials": {
							"api_key": "sample_api_key",
							"service_key": "sample_service_key",
							"teamserver_url": "https://app-agents.contrastsecurity.com/Contrast",
							"username": "username@example.com"
						}
					}]
				}`)
			})

			It("does not double the /Contrast suffix", func() {
				err := contrast.AfterCompile(stager)
				Expect(err).To(BeNil())

				profileDir := filepath.Join(stager.DepDir(), "profile.d")
				files, err := os.ReadDir(profileDir)
				Expect(err).To(BeNil())

				var fileIndex int
				for index, file := range files {
					if strings.Contains(file.Name(), "contrast_security") {
						fileIndex = index
					}
				}

				fileBytes, err := os.ReadFile(filepath.Join(profileDir, files[fileIndex].Name()))
				Expect(err).To(BeNil())

				Expect(string(fileBytes)).To(ContainSubstring(
					"export CONTRAST__API__URL='https://app-agents.contrastsecurity.com/Contrast'",
				))
			})
		})

		Context("Credentials containing special shell characters", func() {
			BeforeEach(func() {
				os.Setenv("VCAP_APPLICATION", `{}`)
				os.Setenv("VCAP_SERVICES", `{
					"contrast-security": [{
						"binding_name": "CCC",
						"credentials": {
							"api_key": "malformedkey$with'special",
							"service_key": "svc key",
							"teamserver_url": "https://example.com",
							"username": "user@example.com"
						}
					}]
				}`)
			})

			It("single-quotes all values in the profile.d script", func() {
				err := contrast.AfterCompile(stager)
				Expect(err).To(BeNil())

				profileDir := filepath.Join(stager.DepDir(), "profile.d")
				files, err := os.ReadDir(profileDir)
				Expect(err).To(BeNil())

				var fileIndex int
				for index, file := range files {
					if strings.Contains(file.Name(), "contrast_security") {
						fileIndex = index
					}
				}

				fileBytes, err := os.ReadFile(filepath.Join(profileDir, files[fileIndex].Name()))
				Expect(err).To(BeNil())

				fileContents := string(fileBytes)
				// single quote escaping: ' → '\''
				Expect(fileContents).To(ContainSubstring("export CONTRAST__API__API_KEY='malformedkey$with'\\''special'"))
				Expect(fileContents).To(ContainSubstring("export CONTRAST__API__SERVICE_KEY='svc key'"))
			})
		})

		Context("Non-string credential value does not panic", func() {
			BeforeEach(func() {
				os.Setenv("VCAP_APPLICATION", `{}`)
				os.Setenv("VCAP_SERVICES", `{
					"contrast-security": [{
						"credentials": {
							"api_key": 12345,
							"service_key": "sample_service_key",
							"teamserver_url": "sample_teamserver_url",
							"username": "username@example.com"
						}
					}]
				}`)
			})
			It("returns empty string for the non-string field and does not panic", func() {
				success, credentials := contrast.GetCredentialsFromEnvironment()
				Expect(success).To(BeTrue())
				Expect(credentials.ApiKey).To(Equal(""))
				Expect(credentials.ServiceKey).To(Equal("sample_service_key"))
			})
		})

		Context("No Contrast Security credentials in VCAP_SERVICES", func() {
			BeforeEach(func() {
				os.Setenv("VCAP_APPLICATION", "{}")
				os.Setenv("VCAP_SERVICES", "{}")
			})

			It("writes the Contrast Security credentials to a file in .profile.d", func() {
				err := contrast.AfterCompile(stager)
				Expect(err).To(BeNil())

				files, err := os.ReadDir(path.Join(stager.BuildDir()))
				Expect(err).To(BeNil())
				Expect(len(files)).To(Equal(0))
			})

		})
	})

	Describe("GetCredentialsFromEnvironment", func() {

		Context("Contrast Security defined in name for user-defined service within VCAP_SERVICES", func() {
			BeforeEach(func() {
				os.Setenv("VCAP_APPLICATION", `{}`)
				os.Setenv("VCAP_SERVICES", `{
                                    "user-provided":[
                                      { "label": "user-provided", 
                                        "name": "contrast-security-service", 
                                        "tags": [ ], 
                                        "instance_name": "sample_instance_name", 
                                        "binding_name": null, 
                                        "credentials": { 
                                          "api_key": "sample_api_key", 
                                          "service_key": "sample_service_key", 
                                          "teamserver_url": "sample_teamserver_url", 
                                          "username": "username@example.com"
                                          }, 
                                          "syslog_drain_url": "", 
                                          "volume_mounts": [ ] 
                                        }
                                      ]
                                    }`)
			})
			It("Returns credentials", func() {
				success, credentials := contrast.GetCredentialsFromEnvironment()
				Expect(success).To(BeTrue())
				Expect(credentials).To(BeEquivalentTo(hooks.ContrastSecurityCredentials{
					ApiKey:      "sample_api_key",
					ServiceKey:  "sample_service_key",
					ContrastUrl: "sample_teamserver_url",
					Username:    "username@example.com",
				}))
			})
		})

		Context("Contrast Security defined in label for user-defined service within VCAP_SERVICES", func() {
			BeforeEach(func() {
				os.Setenv("VCAP_APPLICATION", `{}`)
				os.Setenv("VCAP_SERVICES", `{
                                    "user-provided":[
                                      { "label": "contrast-security-service", 
                                        "name": "sample_service_name", 
                                        "tags": [ ], 
                                        "instance_name": "sample_instance_name", 
                                        "binding_name": null, 
                                        "credentials": { 
                                          "api_key": "sample_api_key", 
                                          "service_key": "sample_service_key", 
                                          "teamserver_url": "sample_teamserver_url", 
                                          "username": "username@example.com"
                                          }, 
                                          "syslog_drain_url": "", 
                                          "volume_mounts": [ ] 
                                        }
                                      ]
                                    }`)
			})
			It("Returns credentials", func() {
				success, credentials := contrast.GetCredentialsFromEnvironment()
				Expect(success).To(BeTrue())
				Expect(credentials).To(BeEquivalentTo(hooks.ContrastSecurityCredentials{
					ApiKey:      "sample_api_key",
					ServiceKey:  "sample_service_key",
					ContrastUrl: "sample_teamserver_url",
					Username:    "username@example.com",
				}))
			})
		})

		Context("Contrast Security defined in tags for user-defined service within VCAP_SERVICES", func() {
			BeforeEach(func() {
				os.Setenv("VCAP_APPLICATION", `{}`)
				os.Setenv("VCAP_SERVICES", `{
                                    "user-provided":[
                                      { "label": "sample_label_name", 
                                        "name": "sample_service_name", 
                                        "tags": [ "contrast-security-service"], 
                                        "instance_name": "sample_instance_name", 
                                        "binding_name": null, 
                                        "credentials": { 
                                          "api_key": "sample_api_key", 
                                          "service_key": "sample_service_key", 
                                          "teamserver_url": "sample_teamserver_url", 
                                          "username": "username@example.com"
                                          }, 
                                          "syslog_drain_url": "", 
                                          "volume_mounts": [ ] 
                                        }
                                      ]
                                    }`)
			})
			It("Returns credentials", func() {
				success, credentials := contrast.GetCredentialsFromEnvironment()
				Expect(success).To(BeTrue())
				Expect(credentials).To(BeEquivalentTo(hooks.ContrastSecurityCredentials{
					ApiKey:      "sample_api_key",
					ServiceKey:  "sample_service_key",
					ContrastUrl: "sample_teamserver_url",
					Username:    "username@example.com",
				}))
			})
		})

		Context("Contrast Security found among multiple service categories in VCAP_SERVICES", func() {
			BeforeEach(func() {
				os.Setenv("VCAP_APPLICATION", `{}`)
				os.Setenv("VCAP_SERVICES", `{
                                      "sample-service":[
                                        { 
                                          "label": "sample-label", 
                                          "provider": null, 
                                          "plan": "sample-plan-name", 
                                          "name": "sample-service-name", 
                                          "tags": [ "Sample Tag" ],
                                          "instance_name": "sample-instance-name", 
                                          "binding_name": null, 
                                          "credentials": { 
                                            "uri": "postgres://example.com", 
                                            "max_conns": "5" 
                                          }, 
                                          "syslog_drain_url": null, 
                                          "volume_mounts": [ ] 
                                        }
                                      ],
                                      "user-provided":[
                                        { 
                                          "label": "user-provided", 
                                          "name": "contrast-security-service", 
                                          "tags": [ ], 
                                          "instance_name": "contrast-security-service", 
                                          "binding_name": null, 
                                          "credentials": { 
                                            "api_key": "sample_api_key", 
                                            "service_key": "sample_service_key", 
                                            "teamserver_url": "sample_teamserver_url", 
                                            "username": "username@example.com" 
                                          }, 
                                          "syslog_drain_url": "", 
                                          "volume_mounts": [ ] 
                                        }
                                      ]}`)
			})
			It("Returns credentials", func() {
				success, credentials := contrast.GetCredentialsFromEnvironment()
				Expect(success).To(BeTrue())
				Expect(credentials).To(BeEquivalentTo(hooks.ContrastSecurityCredentials{
					ApiKey:      "sample_api_key",
					ServiceKey:  "sample_service_key",
					ContrastUrl: "sample_teamserver_url",
					Username:    "username@example.com",
				}))
			})
		})

		Context("Two user-provided services where only the second matches contrast-security", func() {
			BeforeEach(func() {
				os.Setenv("VCAP_APPLICATION", `{}`)
				os.Setenv("VCAP_SERVICES", `{
					"user-provided": [
						{
							"label": "user-provided",
							"name": "my-postgres",
							"tags": [],
							"credentials": {
								"uri": "postgres://example.com"
							}
						},
						{
							"label": "user-provided",
							"name": "contrast-security-service",
							"tags": [],
							"credentials": {
								"api_key": "correct_api_key",
								"service_key": "correct_service_key",
								"teamserver_url": "correct_teamserver_url",
								"username": "correct@example.com"
							}
						}
					]
				}`)
			})
			It("returns credentials from the matching service, not the first in the category", func() {
				success, credentials := contrast.GetCredentialsFromEnvironment()
				Expect(success).To(BeTrue())
				Expect(credentials).To(BeEquivalentTo(hooks.ContrastSecurityCredentials{
					ApiKey:      "correct_api_key",
					ServiceKey:  "correct_service_key",
					ContrastUrl: "correct_teamserver_url",
					Username:    "correct@example.com",
				}))
			})
		})

		Context("Contrast Security undefined for user-defined service within VCAP_SERVICES", func() {
			BeforeEach(func() {
				os.Setenv("VCAP_APPLICATION", `{}`)
				os.Setenv("VCAP_SERVICES", `{
                                    "user-provided":[
                                      { "label": "sample_label_name", 
                                        "name": "sample_service_name", 
                                        "tags": [ ], 
                                        "instance_name": "sample_instance_name", 
                                        "binding_name": null, 
                                        "credentials": { 
                                          "api_key": "sample_api_key", 
                                          "service_key": "sample_service_key", 
                                          "teamserver_url": "sample_teamserver_url", 
                                          "username": "username@example.com"
                                          }, 
                                          "syslog_drain_url": "", 
                                          "volume_mounts": [ ] 
                                        }
                                      ]
                                    }`)
			})
			It("fails but continues", func() {
				success, credentials := contrast.GetCredentialsFromEnvironment()
				Expect(success).To(BeFalse())
				Expect(credentials).To(BeEquivalentTo(hooks.ContrastSecurityCredentials{}))
			})
		})

		Context("No Contrast Security credentials in VCAP_SERVICES", func() {
			BeforeEach(func() {
				os.Setenv("VCAP_APPLICATION", `{}`)
				os.Setenv("VCAP_SERVICES", "{}")
			})

			It("fails but continues", func() {
				success, credentials := contrast.GetCredentialsFromEnvironment()
				Expect(success).To(BeFalse())
				Expect(credentials).To(BeEquivalentTo(hooks.ContrastSecurityCredentials{}))
			})

		})

		Context("No VCAP_SERVICES at all", func() {
			BeforeEach(func() {
				os.Setenv("VCAP_APPLICATION", `{}`)
				os.Unsetenv("VCAP_SERVICES")
			})

			It("fails but continues", func() {
				success, credentials := contrast.GetCredentialsFromEnvironment()
				Expect(success).To(BeFalse())
				Expect(credentials).To(BeEquivalentTo(hooks.ContrastSecurityCredentials{}))
			})

		})

		Context("Malformed VCAP_SERVICES", func() {
			BeforeEach(func() {
				os.Setenv("VCAP_APPLICATION", `{}`)
				os.Setenv("VCAP_SERVICES", "{invalid,json}")
			})

			It("fails but continues", func() {
				success, credentials := contrast.GetCredentialsFromEnvironment()
				Expect(success).To(BeFalse())
				Expect(credentials).To(BeEquivalentTo(hooks.ContrastSecurityCredentials{}))
			})

		})

		Context("Contrast Security credentials in VCAP_SERVICES", func() {
			BeforeEach(func() {
				os.Setenv("VCAP_APPLICATION", `{}`)
				os.Setenv("VCAP_SERVICES", `{
                                                "contrast-security": [
                                                 {
                                                  "binding_name": "CCC",
                                                  "credentials": {
                                                   "api_key": "sample_api_key",
                                                   "org_uuid": "sample_org_uuid",
                                                   "service_key": "sample_service_key",
                                                   "teamserver_url": "sample_teamserver_url",
                                                   "username": "username@example.com"
                                                  }
                                                 }
                                                ]
                                               }`)
			})

			It("Returns credentials", func() {
				success, credentials := contrast.GetCredentialsFromEnvironment()
				Expect(success).To(BeTrue())
				Expect(credentials).To(BeEquivalentTo(hooks.ContrastSecurityCredentials{
					ApiKey:      "sample_api_key",
					OrgUuid:     "sample_org_uuid",
					ServiceKey:  "sample_service_key",
					ContrastUrl: "sample_teamserver_url",
					Username:    "username@example.com",
				}))
			})

		})
	})
})
