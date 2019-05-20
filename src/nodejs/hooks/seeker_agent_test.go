package hooks_test

import (
	"bytes"
	"fmt"
	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/nodejs-buildpack/src/nodejs/hooks"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/jarcoal/httpmock.v1"
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
		Expect(err).To(BeNil())

		depsDir, err = ioutil.TempDir("", "nodejs-buildpack.deps.")
		Expect(err).To(BeNil())

		depsIdx = "07"
		err = os.MkdirAll(filepath.Join(depsDir, depsIdx), 0755)

		buffer = new(bytes.Buffer)
		logger = libbuildpack.NewLogger(buffer)

		logger := libbuildpack.NewLogger(os.Stdout)
		command := &libbuildpack.Command{}

		seeker = hooks.SeekerAfterCompileHook{
			Command: command,
			Log:     logger,
		}
		httpmock.Reset()
	})

	JustBeforeEach(func() {
		args := []string{buildDir, "", depsDir, depsIdx}
		stager = libbuildpack.NewStager(args, logger, &libbuildpack.Manifest{})
	})

	AfterEach(func() {

		err = os.RemoveAll(buildDir)
		Expect(err).To(BeNil())

		err = os.RemoveAll(depsDir)
		Expect(err).To(BeNil())
	})
	Describe("AfterCompile - obtain agent by extracting from sensor", func() {
		var (
			oldVcapApplication string
			oldVcapServices    string
			oldBpDebug         string
		)
		BeforeEach(func() {
			oldVcapApplication = os.Getenv("VCAP_APPLICATION")
			oldVcapServices = os.Getenv("VCAP_SERVICES")
			oldBpDebug = os.Getenv("BP_DEBUG")
			mockSeekerVersionThatSupportsOnlySensorDownload()
			mockSensorDownload()

		})
		AfterEach(func() {
			assertSensorDownload()
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
							"seeker_server_url": "http://10.120.9.117:9911",
					       "enterprise_server_url": "http://10.120.9.117:8082",
					       "sensor_host": "localhost",
					       "sensor_port": "9911"
					     },
					     "syslog_drain_url": "",
					     "volume_mounts": [],
					     "label": "user-provided",
					     "tags": []
					   }
					 ]
					}`)
			})
			It("installs seeker", func() {
				err = seeker.AfterCompile(stager)
				Expect(err).To(BeNil())

				// Sets up profile.d
				envFile := filepath.Join(depsDir, depsIdx, "profile.d", "seeker-env.sh")
				contents, err := ioutil.ReadFile(envFile)
				Expect(err).To(BeNil())

				expected := "\n" +
					"export SEEKER_SENSOR_HOST=localhost\n" +
					"export SEEKER_SENSOR_HTTP_PORT=9911\n" +
					"export SEEKER_SERVER_URL=http://10.120.9.117:9911\n"
				Expect(string(contents)).To(Equal(expected))
			})
		})
		Context("VCAP_SERVICES contains seeker service - as a regular service", func() {
			BeforeEach(func() {
				os.Setenv("VCAP_APPLICATION", `{"name":"pcf app"}`)
				os.Setenv("VCAP_SERVICES", `{
													"seeker-security-service": [
													 {
													   "name": "seeker_instace",
													   "instance_name": "seeker_instace",
													   "binding_name": null,
													   "credentials": {
													     "sensor_host": "localhost",
													     "sensor_port": "9911",
														"seeker_server_url": "http://10.120.9.117:9911",
					       								"enterprise_server_url": "http://10.120.9.117:8082"

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
			It("installs seeker", func() {
				err = seeker.AfterCompile(stager)
				Expect(err).To(BeNil())

				// Sets up profile.d
				contents, err := ioutil.ReadFile(filepath.Join(depsDir, depsIdx, "profile.d", "seeker-env.sh"))
				Expect(err).To(BeNil())
				expected := "\n" +
					"export SEEKER_SENSOR_HOST=localhost\n" +
					"export SEEKER_SENSOR_HTTP_PORT=9911\n" +
					"export SEEKER_SERVER_URL=http://10.120.9.117:9911\n"
				Expect(string(contents)).To(Equal(expected))
			})

		})
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
			mockSeekerVersionThatSupportsAgentDownload()
			mockAgentDownload()
		})
		AfterEach(func() {
			assertAgentDownload()
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
															"seeker_server_url": "http://10.120.9.117:9911",
													       "enterprise_server_url": "http://10.120.9.117:8082",
													       "sensor_host": "localhost",
													       "sensor_port": "9911"
													     },
													     "syslog_drain_url": "",
													     "volume_mounts": [],
													     "label": "user-provided",
													     "tags": []
													   }
													 ]
													}`)
			})
			It("installs seeker", func() {
				err = seeker.AfterCompile(stager)
				Expect(err).To(BeNil())

				// Sets up profile.d
				contents, err := ioutil.ReadFile(filepath.Join(depsDir, depsIdx, "profile.d", "seeker-env.sh"))
				Expect(err).To(BeNil())

				expected := "\n" +
					"export SEEKER_SENSOR_HOST=localhost\n" +
					"export SEEKER_SENSOR_HTTP_PORT=9911\n" +
					"export SEEKER_SERVER_URL=http://10.120.9.117:9911\n"
				Expect(string(contents)).To(Equal(expected))
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
													     "sensor_host": "localhost",
													     "sensor_port": "9911",
														"seeker_server_url": "http://10.120.9.117:9911",
												       "enterprise_server_url": "http://10.120.9.117:8082"

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
			It("installs seeker", func() {
				err = seeker.AfterCompile(stager)
				Expect(err).To(BeNil())

				// Sets up profile.d
				contents, err := ioutil.ReadFile(filepath.Join(depsDir, depsIdx, "profile.d", "seeker-env.sh"))
				Expect(err).To(BeNil())

				expected := "\n" +
					"export SEEKER_SENSOR_HOST=localhost\n" +
					"export SEEKER_SENSOR_HTTP_PORT=9911\n" +
					"export SEEKER_SERVER_URL=http://10.120.9.117:9911\n"
				Expect(string(contents)).To(Equal(expected))
			})

		})

	})
	Describe("AfterCompile - agent download choosing strategy", func() {
		var (
			oldVcapApplication string
			oldVcapServices    string
			oldBpDebug         string
		)
		BeforeEach(func() {
			oldVcapApplication = os.Getenv("VCAP_APPLICATION")
			oldVcapServices = os.Getenv("VCAP_SERVICES")
			oldBpDebug = os.Getenv("BP_DEBUG")
			os.Unsetenv("SEEKER_APP_ENTRY_POINT")
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
														"seeker_server_url": "http://10.120.9.117:9911",
					       								"enterprise_server_url": "http://10.120.9.117:8082",
												        "sensor_host": "localhost",
												        "sensor_port": "9911"
												      },
												      "syslog_drain_url": "",
												      "volume_mounts": [],
												      "label": "user-provided",
												      "tags": []
												    }
												  ]
												 }`)
			})
			It("Chooses downloading the sensor for Seeker versions older than 2018.05 (including 2018.05)", func() {
				mockSensorDownload()
				seekerVersionSupportingSensorDownloadOnly := []string{"2018.05", "2018.04", "2018.03", "2018.02", "2018.01", "2017.12", "2017.11", "2017.10", "2017.09", "2017.08", "2017.05", "2017.04", "2017.03", "2017.02", "2017.01"}
				for _, seekerVersion := range seekerVersionSupportingSensorDownloadOnly {
					mockSpecificSeekerVersion(seekerVersion)
					err = seeker.AfterCompile(stager)
					Expect(err).To(BeNil())
				}
				sensorDownloadCount := httpmock.GetCallCountInfo()["GET "+getSensorURL()]
				agentDownloadCount := httpmock.GetCallCountInfo()["GET "+getAgentURL()]
				Expect(sensorDownloadCount).To(Equal(len(seekerVersionSupportingSensorDownloadOnly)))
				Expect(agentDownloadCount).To(Equal(0))
			})
			It("Chooses downloading the agent for Seeker versions newer than 2018.05", func() {
				mockAgentDownload()
				seekerVersionSupportingAgentDownload := []string{"2018.06", "2018.07", "2018.08", "2018.09", "2018.10", "2018.11", "2018.12", "2019.01", "2019.02", "2019.03", "2019.04", "2019.05"}
				for _, seekerVersion := range seekerVersionSupportingAgentDownload {
					mockSpecificSeekerVersion(seekerVersion)
					err = seeker.AfterCompile(stager)
					Expect(err).To(BeNil())
				}
				sensorDownloadCount := httpmock.GetCallCountInfo()["GET "+getSensorURL()]
				agentDownloadCount := httpmock.GetCallCountInfo()["GET "+getAgentURL()]
				Expect(agentDownloadCount).To(Equal(len(seekerVersionSupportingAgentDownload)))
				Expect(sensorDownloadCount).To(Equal(0))
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
			mockSeekerVersionThatSupportsAgentDownload()
			mockAgentDownload()
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
														"seeker_server_url": "http://10.120.9.117:9911",
												       "enterprise_server_url": "http://10.120.9.117:8082",
												        "sensor_host": "localhost",
												        "sensor_port": "9911"
												      },
												      "syslog_drain_url": "",
												      "volume_mounts": [],
												      "label": "user-provided",
												      "tags": []
												    }
												  ]
												 }`)
			})
			It("Prepends the require to the server.js file", func() {
				entryPointPath := filepath.Join(buildDir, "server.js")
				Expect(entryPointPath).ToNot(BeAnExistingFile())
				Expect(ioutil.WriteFile(entryPointPath, []byte("some mock javascript code"), 0755)).To(Succeed())
				err = seeker.AfterCompile(stager)
				Expect(err).To(BeNil())
				contents, err := ioutil.ReadFile(entryPointPath)
				Expect(err).To(BeNil())
				Expect(string(contents)).To(Equal(
					"require('./seeker/node_modules/@synopsys-sig/seeker-inline');\n" +
						"some mock javascript code\n"))
			})
			It("Fails to prepend the require to the server.js file - when the file does not exist", func() {
				entryPointPath := filepath.Join(buildDir, "server.js")
				Expect(entryPointPath).ToNot(BeAnExistingFile())
				err = seeker.AfterCompile(stager)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("no such file or directory"))
			})

		})
	})
})

func assertSensorDownload() {
	Expect(httpmock.GetCallCountInfo()["GET "+getVersionURL()]).To(Equal(1))
	Expect(httpmock.GetCallCountInfo()["GET "+getSensorURL()]).To(Equal(1))
	Expect(httpmock.GetCallCountInfo()).ShouldNot(ContainElement("GET " + getAgentURL()))
	Expect(httpmock.GetTotalCallCount()).To(Equal(2))
}
func assertAgentDownload() {
	Expect(httpmock.GetCallCountInfo()["GET "+getVersionURL()]).To(Equal(1))
	Expect(httpmock.GetCallCountInfo()["GET "+getAgentURL()]).To(Equal(1))
	Expect(httpmock.GetCallCountInfo()).ShouldNot(ContainElement("GET " + getSensorURL()))
	Expect(httpmock.GetTotalCallCount()).To(Equal(2))
}

func getAgentZip() ([]byte, error) {
	return getFixtureContent("NODEJS_agent.zip")
}
func getSensorZip() ([]byte, error) {
	return getFixtureContent("NODEJS_SensorWithAgent.zip")
}
func getFixtureContent(fileName string) ([]byte, error) {
	path, err := filepath.Abs("../../../fixtures/seeker/" + fileName)
	if err != nil {
		return nil, err
	}
	fileContent, err := ioutil.ReadFile(path)
	return fileContent, err
}

func mockSeekerVersionThatSupportsOnlySensorDownload() {
	httpmock.RegisterResponder("GET", getVersionURL(),
		httpmock.NewStringResponder(200, `{"publicName":"Seeker Enterprise Server","version":"2018.05-SP1-SNAPSHOT","buildNumber":"20180629131550","scmBranch":"origin/release/v2018.06","scmRevision":"815ba309"}`))
}

func mockSeekerVersionThatSupportsAgentDownload() {
	httpmock.RegisterResponder("GET", getVersionURL(),
		httpmock.NewStringResponder(200, `{"publicName":"Seeker Enterprise Server","version":"2018.06-SNAPSHOT","buildNumber":"20180629131550","scmBranch":"origin/release/v2018.06","scmRevision":"815ba309"}`))
}
func mockSensorDownload() {
	zipContent, _ := getSensorZip()
	httpmock.RegisterResponder("GET", getSensorURL(),
		httpmock.NewBytesResponder(200, zipContent))
}

func mockAgentDownload() {
	zipContent, _ := getAgentZip()
	httpmock.RegisterResponder("GET", getAgentURL(),
		httpmock.NewBytesResponder(200, zipContent))
}
func getSensorURL() string {
	return "http://10.120.9.117:8082/rest/ui/installers/binaries/LINUX"
}
func getAgentURL() string {
	return "http://10.120.9.117:8082/rest/ui/installers/agents/binaries/NODEJS"
}
func getVersionURL() string {
	return "http://10.120.9.117:8082/rest/api/version"
}
func mockSpecificSeekerVersion(seekerVersion string) {
	jsonVersionPayload := fmt.Sprintf(`{"publicName":"Seeker Enterprise Server","version":"%s","buildNumber":"20180629131550","scmBranch":"origin/release/v2018.06","scmRevision":"815ba309"}`, seekerVersion)
	httpmock.RegisterResponder("GET", getVersionURL(),
		httpmock.NewStringResponder(200, jsonVersionPayload))
}
