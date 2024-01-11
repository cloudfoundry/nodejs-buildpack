package hooks_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/nodejs-buildpack/src/nodejs/hooks"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type Command struct {
	called bool
	args   []string
}

func (c *Command) Execute(dir string, stdout io.Writer, stderr io.Writer, program string, args ...string) error {
	c.called = true
	c.args = args
	return nil
}

// MockHttpClient is a mock implementation of the HTTPClient interface for testing.
type MockHttpClient struct {
	Response string
	Error    string
}

func (m *MockHttpClient) Get(url string) (*http.Response, error) {
	if m.Error == "" {
		reader := strings.NewReader(m.Response)
		return &http.Response{Body: io.NopCloser(reader)}, nil
	} else {
		return nil, errors.New(m.Error)
	}

}

var _ = Describe("Sealights hook", func() {
	var (
		err         error
		buildDir    string
		logger      *libbuildpack.Logger
		buffer      *bytes.Buffer
		stager      *libbuildpack.Stager
		sealights   *hooks.SealightsHook
		yamlFile    *libbuildpack.YAML
		build       string
		proxy       string
		labId       string
		projectRoot string
		testStage   string
		// procfile                      string
		command                       *Command
		httpClient                    *MockHttpClient
		procfileName                  = "Procfile"
		packageJsonName               = "package.json"
		manifestName                  = "manifest.yml"
		originalStartCommand          = "node index.js --build 192 --name Good"
		testProcfile                  = "web: " + originalStartCommand
		testPackageJson               = "{\n    \"scripts\": {\n        \"start\": \"" + originalStartCommand + "\"\n    }\n}"
		testPackageJsonWithoutScripts = "{\n    \"skriptz\": {\n        \"start\": \"" + originalStartCommand + "\"\n    }\n}"
		testPackageJsonWithoutStart   = "{\n    \"scripts\": {\n        \"begin\": \"" + originalStartCommand + "\"\n    }\n}"
		testManifest                  = "---\n" +
			"applications:\n" +
			"  - name: Good\n" +
			"    command: " + originalStartCommand
		expected         = strings.ReplaceAll("./node_modules/.bin/slnodejs run  --useinitialcolor true --token good_token --buildsessionid goodBsid --proxy http://localhost:1886 --labid Roni's --projectroot project/root --teststage \"Unit Tests\" index.js --build 192 --name Good", " ", "")
		expectedWithFile = strings.ReplaceAll("./node_modules/.bin/slnodejs run  --useinitialcolor true --token good_token --buildsessionidfile build/id/file --proxy http://localhost:1886 --labid Roni's --projectroot project/root --teststage \"Unit Tests\" index.js --build 192 --name Good", " ", "")
	)

	BeforeEach(func() {
		buildDir, err = os.MkdirTemp("", "nodejs-buildpack.build.")
		Expect(err).To(BeNil())

		buffer = new(bytes.Buffer)
		logger = libbuildpack.NewLogger(buffer)
		args := []string{buildDir, ""}
		stager = libbuildpack.NewStager(args, logger, &libbuildpack.Manifest{})

		build = os.Getenv("SL_BUILD_SESSION_ID_FILE")
		proxy = os.Getenv("SL_PROXY")
		labId = os.Getenv("SL_LAB_ID")
		projectRoot = os.Getenv("SL_PROJECT_ROOT")
		testStage = os.Getenv("SL_TEST_STAGE")
		command = &Command{}
		httpClient = &MockHttpClient{
			Response: "",
			Error:    "",
		}
		sealights = hooks.NewSealightsHook(logger, command, httpClient)
	})

	AfterEach(func() {
		err = os.Setenv("SL_BUILD_SESSION_ID", build)
		Expect(err).To(BeNil())
		err = os.Setenv("SL_PROXY", proxy)
		Expect(err).To(BeNil())
		err = os.Setenv("SL_LAB_ID", labId)
		Expect(err).To(BeNil())
		err = os.Setenv("SL_PROJECT_ROOT", projectRoot)
		Expect(err).To(BeNil())
		err = os.Setenv("SL_TEST_STAGE", testStage)
		Expect(err).To(BeNil())
		err = os.Unsetenv("VCAP_SERVICES")
		Expect(err).To(BeNil())
		// err = os.WriteFile(filepath.Join(stager.BuildDir(), procfileName), []byte(procfile), 0755)
		// Expect(err).To(BeNil())
		err = os.RemoveAll(buildDir)
		Expect(err).To(BeNil())
	})

	Describe("AfterCompile", func() {
		var (
			token    = "good_token"
			bsid     = "goodBsid"
			bsidFile = "build/id/file"
			proxy    = "http://localhost:1886"
			lab      = "Roni's"
			root     = "project/root"
			stage    = "Unit Tests"
		)
		BeforeEach(func() {
			Expect(err).To(BeNil())
			err = os.Setenv("SL_BUILD_SESSION_ID", bsid)
			Expect(err).To(BeNil())
			err = os.Setenv("SL_BUILD_SESSION_ID_FILE", bsidFile)
			Expect(err).To(BeNil())
			err = os.Setenv("SL_PROXY", proxy)
			Expect(err).To(BeNil())
		})
		Context("Sealigts not injected well", func() {
			BeforeEach(func() {
				err = os.WriteFile(filepath.Join(stager.BuildDir(), procfileName), []byte(testProcfile), 0755)
				Expect(err).To(BeNil())
			})
			It("Not found in VCAP_Services", func() {
				with := sealights.RunWithSealights()
				Expect(with).To(BeFalse())
				err = sealights.AfterCompile(stager)
				Expect(err).To(BeNil())
				Expect(command.called).To(BeFalse())
			})
			It("hook fails with empty token", func() {
				err = os.Setenv("VCAP_SERVICES", `{"user-provided":[
														{ "label": "user-provided",
															"name": "sealights",
															"credentials": {
															"token": ""
															}
															}
													    ]}`)

				Expect(err).To(BeNil())
				err = sealights.AfterCompile(stager)
				Expect(err).To(BeNil())
				Expect(command.called).To(BeFalse())
			})
		})

		Context("Configure sealight with VCAP_SERVICES only", func() {
			BeforeEach(func() {
				Expect(err).To(BeNil())
				err = os.Setenv("SL_BUILD_SESSION_ID", "")
				Expect(err).To(BeNil())
				err = os.Setenv("SL_BUILD_SESSION_ID_FILE", "")
				Expect(err).To(BeNil())
				err = os.Setenv("SL_PROXY", "")
				Expect(err).To(BeNil())
				err = os.Setenv("SL_LAB_ID", "")
				Expect(err).To(BeNil())
				err = os.Setenv("SL_PROJECT_ROOT", "")
				Expect(err).To(BeNil())
				err = os.Setenv("SL_TEST_STAGE", "")

				err = os.WriteFile(filepath.Join(stager.BuildDir(), procfileName), []byte(testProcfile), 0755)
				Expect(err).To(BeNil())
			})

			It("hook fails with empty token and tokenFile", func() {
				vcapTemplate := `{
					"user-provided":[{
						"label": "user-provided",
						"name": "sealights",
						"credentials": {
							"token": "",
							"tokenFile": "",
							"buildSessionId": "` + bsid + `"
						}
					}]
				}`
				err = os.Setenv("VCAP_SERVICES", vcapTemplate)
				Expect(err).To(BeNil())

				err = sealights.AfterCompile(stager)
				Expect(err).To(BeNil())

				Expect(command.called).To(BeFalse())
				bytes, err := os.ReadFile(filepath.Join(stager.BuildDir(), procfileName))
				Expect(err).To(BeNil())
				cleanResult := strings.ReplaceAll(string(bytes), " ", "")
				expectedCommand := strings.ReplaceAll("node index.js --build 192 --name Good", " ", "")
				Expect(cleanResult).To(Equal("web:" + expectedCommand))
			})

			It("hook fails with empty buildSessionId and buildSessionIdFile", func() {
				vcapTemplate := `{
					"user-provided":[{
						"label": "user-provided",
						"name": "sealights",
						"credentials": {
							"token": "` + token + `",
							"buildSessionId": "",
							"buildSessionIdFile": ""
						}
					}]
				}`
				err = os.Setenv("VCAP_SERVICES", vcapTemplate)
				Expect(err).To(BeNil())

				err = sealights.AfterCompile(stager)
				Expect(err).To(BeNil())

				Expect(command.called).To(BeFalse())
				bytes, err := os.ReadFile(filepath.Join(stager.BuildDir(), procfileName))
				Expect(err).To(BeNil())
				cleanResult := strings.ReplaceAll(string(bytes), " ", "")
				expectedCommand := strings.ReplaceAll("node index.js --build 192 --name Good", " ", "")
				Expect(cleanResult).To(Equal("web:" + expectedCommand))
			})

			It("hook doesn't fail with token or tokenFile and buildSessionId or buildSessionIdFile", func() {
				vcapTemplate := `{
					"user-provided":[{
						"label": "user-provided",
						"name": "sealights",
						"credentials": {
							"token": "` + token + `",
							"buildSessionId": "` + bsid + `"
						}
					}]
				}`
				err = os.Setenv("VCAP_SERVICES", vcapTemplate)
				Expect(err).To(BeNil())

				err = sealights.AfterCompile(stager)
				Expect(err).To(BeNil())
				Expect(command.called).To(BeTrue())
				bytes, err := os.ReadFile(filepath.Join(stager.BuildDir(), procfileName))
				Expect(err).To(BeNil())
				cleanResult := strings.ReplaceAll(string(bytes), " ", "")
				expectedCommand := strings.ReplaceAll("./node_modules/.bin/slnodejs run --useinitialcolor true --token good_token --buildsessionid goodBsid index.js --build 192 --name Good", " ", "")
				Expect(cleanResult).To(Equal("web:" + expectedCommand))
			})

			It("hook pass all params to updated cli command", func() {
				vcapTemplate := `{
					"user-provided":[{
						"label": "user-provided",
						"name": "sealights",
						"credentials": {
							"token": "` + token + `",
							"buildSessionId": "` + bsid + `",
							"labId": "goodLab",
							"projectRoot": "./",
							"testStage": "Good tests",
							"proxy": "goodProxy",
							"proxyUsername": "goodName",
							"proxyPassword": "goodPassword"
						}
					}]
				}`
				err = os.Setenv("VCAP_SERVICES", vcapTemplate)
				Expect(err).To(BeNil())

				err = sealights.AfterCompile(stager)
				Expect(err).To(BeNil())
				Expect(command.called).To(BeTrue())
				bytes, err := os.ReadFile(filepath.Join(stager.BuildDir(), procfileName))
				Expect(err).To(BeNil())
				cleanResult := strings.ReplaceAll(string(bytes), " ", "")
				expectedCommand := strings.ReplaceAll("./node_modules/.bin/slnodejs run --useinitialcolor true --token good_token --buildsessionid goodBsid --proxy goodProxy --proxyUsername goodName --proxyPassword goodPassword --labid goodLab --projectroot ./ --teststage \"Good tests\" index.js --build 192 --name Good", " ", "")
				Expect(cleanResult).To(Equal("web:" + expectedCommand))
			})
		})

		Context("Sealights injection", func() {
			BeforeEach(func() {
				os.Setenv("VCAP_SERVICES", `{"user-provided":[
														{ 
															"label": "user-provided",
															"name": "sealights",
															"credentials": {
																"token": "`+token+`"
															}
															}
													    ]}`)
			})
			Context("build new application run command in Procfile", func() {
				BeforeEach(func() {
					err = os.WriteFile(filepath.Join(stager.BuildDir(), procfileName), []byte(testProcfile), 0755)
					Expect(err).To(BeNil())
				})
				It("test application run cmd creation from bsid file", func() {
					err = os.Setenv("SL_LAB_ID", lab)
					Expect(err).To(BeNil())
					err = os.Setenv("SL_PROXY", proxy)
					Expect(err).To(BeNil())
					err = os.Setenv("SL_PROJECT_ROOT", root)
					Expect(err).To(BeNil())
					err = os.Setenv("SL_TEST_STAGE", stage)
					Expect(err).To(BeNil())
					err = sealights.SetApplicationStartInProcfile(stager)
					Expect(err).To(BeNil())
					bytes, err := os.ReadFile(filepath.Join(stager.BuildDir(), procfileName))
					Expect(err).To(BeNil())
					cleanResult := strings.ReplaceAll(string(bytes), " ", "")
					Expect(cleanResult).To(Equal("web:" + expectedWithFile))
				})
				It("hook fails with empty build session id", func() {
					err = os.Setenv("SL_BUILD_SESSION_ID", "")
					Expect(err).NotTo(HaveOccurred())
					err = os.Setenv("SL_BUILD_SESSION_ID_FILE", "")
					Expect(err).NotTo(HaveOccurred())
					err = sealights.SetApplicationStartInProcfile(stager)
					Expect(err).To(MatchError(ContainSubstring(hooks.EmptyBuildError)))
				})
				It("test application run cmd creation", func() {
					err = os.Setenv("SL_LAB_ID", lab)
					Expect(err).To(BeNil())
					err = os.Setenv("SL_PROXY", proxy)
					Expect(err).To(BeNil())
					err = os.Setenv("SL_PROJECT_ROOT", root)
					Expect(err).To(BeNil())
					err = os.Setenv("SL_TEST_STAGE", stage)
					Expect(err).To(BeNil())
					err = os.Setenv("SL_BUILD_SESSION_ID_FILE", "")
					Expect(err).NotTo(HaveOccurred())
					Expect(err).To(BeNil())
					err = sealights.SetApplicationStartInProcfile(stager)
					bytes, err := os.ReadFile(filepath.Join(stager.BuildDir(), procfileName))
					Expect(err).To(BeNil())
					cleanResult := strings.ReplaceAll(string(bytes), " ", "")
					Expect(cleanResult).To(Equal("web:" + expected))
				})
			})

			Context("fail to update package.json scripts", func() {
				BeforeEach(func() {
					err = os.WriteFile(filepath.Join(stager.BuildDir(), packageJsonName), []byte(testPackageJsonWithoutScripts), 0755)
					Expect(err).To(BeNil())
				})

				AfterEach(func() {
					os.Remove(filepath.Join(stager.BuildDir(), packageJsonName))
				})

				It("fail to find scripts section in package.json", func() {
					err = sealights.SetApplicationStartInPackageJson(stager)
					Expect(err).ShouldNot(BeNil())
				})
			})

			Context("fail to update package.json start", func() {
				BeforeEach(func() {
					err = os.WriteFile(filepath.Join(stager.BuildDir(), packageJsonName), []byte(testPackageJsonWithoutStart), 0755)
					Expect(err).To(BeNil())
				})

				AfterEach(func() {
					os.Remove(filepath.Join(stager.BuildDir(), packageJsonName))
				})

				It("fail to start under scripts section in package.json", func() {
					err = sealights.SetApplicationStartInPackageJson(stager)
					Expect(err).ShouldNot(BeNil())
				})
			})

			Context("build new application run command in package.json", func() {
				BeforeEach(func() {
					err = os.WriteFile(filepath.Join(stager.BuildDir(), packageJsonName), []byte(testPackageJson), 0755)
					Expect(err).To(BeNil())
				})

				It("test application run cmd creation from bsid file", func() {
					err = os.Setenv("SL_LAB_ID", lab)
					Expect(err).To(BeNil())
					err = os.Setenv("SL_PROJECT_ROOT", root)
					Expect(err).To(BeNil())
					err = os.Setenv("SL_TEST_STAGE", stage)
					Expect(err).To(BeNil())
					err = sealights.SetApplicationStartInPackageJson(stager)
					Expect(err).To(BeNil())
					packageJson, err := sealights.ReadPackageJson(stager)
					Expect(err).To(BeNil())
					cleanResult := strings.ReplaceAll(packageJson["scripts"].(map[string]interface{})["start"].(string), " ", "")
					Expect(cleanResult).To(Equal(expectedWithFile))
				})
				It("hook fails with empty build session id", func() {
					err = os.Setenv("SL_BUILD_SESSION_ID", "")
					Expect(err).NotTo(HaveOccurred())
					err = os.Setenv("SL_BUILD_SESSION_ID_FILE", "")
					Expect(err).NotTo(HaveOccurred())
					err = sealights.SetApplicationStartInPackageJson(stager)
					Expect(err).To(MatchError(ContainSubstring(hooks.EmptyBuildError)))
				})
				It("test application run cmd creation", func() {
					err = os.Setenv("SL_LAB_ID", lab)
					Expect(err).To(BeNil())
					err = os.Setenv("SL_PROJECT_ROOT", root)
					Expect(err).To(BeNil())
					err = os.Setenv("SL_TEST_STAGE", stage)
					Expect(err).To(BeNil())
					err = os.Setenv("SL_BUILD_SESSION_ID_FILE", "")
					Expect(err).NotTo(HaveOccurred())
					Expect(err).To(BeNil())
					err = sealights.SetApplicationStartInPackageJson(stager)
					packageJson, err := sealights.ReadPackageJson(stager)
					Expect(err).To(BeNil())
					cleanResult := strings.ReplaceAll(packageJson["scripts"].(map[string]interface{})["start"].(string), " ", "")
					Expect(cleanResult).To(Equal(expected))
				})
			})

			Context("build new application run command in manifest", func() {
				BeforeEach(func() {
					err = os.WriteFile(filepath.Join(stager.BuildDir(), manifestName), []byte(testManifest), 0755)
					Expect(err).To(BeNil())
				})

				It("test application run cmd creation from bsid file", func() {
					err = os.Setenv("SL_LAB_ID", lab)
					Expect(err).To(BeNil())
					err = os.Setenv("SL_PROJECT_ROOT", root)
					Expect(err).To(BeNil())
					err = os.Setenv("SL_TEST_STAGE", stage)
					Expect(err).To(BeNil())
					err = sealights.SetApplicationStartInManifest(stager)
					Expect(err).To(BeNil())
					err, manifestFile := sealights.ReadManifestFile(stager, yamlFile)
					Expect(err).To(BeNil())
					cleanResult := strings.ReplaceAll(manifestFile.Applications[0].Command, " ", "")
					Expect(cleanResult).To(Equal(expectedWithFile))
				})
				It("hook fails with empty build session id", func() {
					err = os.Setenv("SL_BUILD_SESSION_ID", "")
					Expect(err).NotTo(HaveOccurred())
					err = os.Setenv("SL_BUILD_SESSION_ID_FILE", "")
					Expect(err).NotTo(HaveOccurred())
					err = sealights.SetApplicationStartInManifest(stager)
					Expect(err).To(MatchError(ContainSubstring(hooks.EmptyBuildError)))
				})
				It("test application run cmd creation", func() {
					err = os.Setenv("SL_LAB_ID", lab)
					Expect(err).To(BeNil())
					err = os.Setenv("SL_PROJECT_ROOT", root)
					Expect(err).To(BeNil())
					err = os.Setenv("SL_TEST_STAGE", stage)
					Expect(err).To(BeNil())
					err = os.Setenv("SL_BUILD_SESSION_ID_FILE", "")
					Expect(err).NotTo(HaveOccurred())
					Expect(err).To(BeNil())
					err = sealights.SetApplicationStartInManifest(stager)
					err, manifestFile := sealights.ReadManifestFile(stager, yamlFile)
					Expect(err).To(BeNil())
					cleanResult := strings.ReplaceAll(manifestFile.Applications[0].Command, " ", "")
					Expect(cleanResult).To(Equal(expected))
				})
			})
		})

		Context("Sealights agent installation", func() {
			customUrl := "customUrl"
			customVersion := "customVersion"
			recommendedVersion := "goodVersion"

			setVcap := func(version string, customAgentUrl string) {
				vcapTemplate := `{
					"user-provided":[{
						"label": "user-provided",
						"name": "sealights",
						"credentials": {
							"token": "%s",
							"version": "%s",
							"customAgentUrl": "%s"
						}
					}]
				}`

				os.Setenv("VCAP_SERVICES", fmt.Sprintf(vcapTemplate, token, version, customAgentUrl))
			}

			BeforeEach(func() {
				err = os.Setenv("SL_DOMAIN", "my-domain")
				Expect(err).To(BeNil())
				err = os.WriteFile(filepath.Join(stager.BuildDir(), procfileName), []byte(testProcfile), 0755)
				Expect(err).To(BeNil())
			})

			It("get recomended version from server", func() {
				setVcap("", "")
				httpClient.Response = `{"agent":{"version": "` + recommendedVersion + `"}}`

				err = sealights.AfterCompile(stager)

				Expect(err).To(BeNil())
				Expect(command.called).To(Equal(true))
				Expect(command.args[1]).To(Equal("slnodejs@" + recommendedVersion))
			})
			It("shouldn't get recomended version from server if SL_DOMAIN not set", func() {
				err = os.Setenv("SL_DOMAIN", "")
				Expect(err).To(BeNil())
				setVcap("", "")
				httpClient.Response = `{"agent":{"version": "` + recommendedVersion + `"}}`

				err = sealights.AfterCompile(stager)

				Expect(err).To(BeNil())
				Expect(command.called).To(Equal(true))
				Expect(command.args[1]).To(Equal("slnodejs@latest"))
			})
			It("install default version if no other provided and get recomended version failed", func() {
				setVcap("", "")
				httpClient.Error = "some error"

				err = sealights.AfterCompile(stager)

				Expect(err).To(BeNil())
				Expect(command.args[1]).To(Equal("slnodejs@latest"))
			})
			It("use custom url if provided", func() {
				setVcap("", customUrl)

				err = sealights.AfterCompile(stager)

				Expect(err).To(BeNil())
				Expect(command.args[1]).To(Equal(customUrl))
			})
			It("should not get custom version if customUrl provided", func() {
				setVcap(customVersion, customUrl)

				err = sealights.AfterCompile(stager)

				Expect(err).To(BeNil())
				Expect(command.args[1]).To(Equal(customUrl))
			})
			It("use version parameter if provided", func() {
				setVcap(customVersion, "")

				err = sealights.AfterCompile(stager)

				Expect(err).To(BeNil())
				Expect(command.args[1]).To(Equal("slnodejs@" + customVersion))
			})
			It("should not get recomended version from server if customVersion provided", func() {
				setVcap(customVersion, "")

				httpClient.Response = `{"agent":{"version": "` + recommendedVersion + `"}}`

				err = sealights.AfterCompile(stager)

				Expect(err).To(BeNil())
				Expect(command.args[1]).To(Equal("slnodejs@" + customVersion))
			})
		})
	})
})
