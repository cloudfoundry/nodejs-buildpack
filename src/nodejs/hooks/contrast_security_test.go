package hooks_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path"

	"github.com/cloudfoundry/libbuildpack"

	"github.com/cloudfoundry/nodejs-buildpack/src/nodejs/hooks"

	. "github.com/onsi/ginkgo"
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
			tmpDir, _ := ioutil.TempDir("", "contrast_security_test")
			args := []string{tmpDir, "", ".", ""}
			stager = libbuildpack.NewStager(args, logger, &libbuildpack.Manifest{})
		})

		AfterEach(func() {
			os.RemoveAll(stager.BuildDir())
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

			It("writes the Contrast Security credentials to a file in .profile.d", func() {
				err := contrast.AfterCompile(stager)
				Expect(err).To(BeNil())

				files, err := ioutil.ReadDir(path.Join(stager.BuildDir(), ".profile.d"))
				Expect(err).To(BeNil())
				Expect(len(files)).To(Equal(1))

				file := files[0]
				Expect(file.Name()).To(Equal("contrast_security"))

				fileBytes, err := ioutil.ReadFile(path.Join(stager.BuildDir(), ".profile.d", file.Name()))

				if err != nil {
					Fail(err.Error())
				}

				fileContents := string(fileBytes)
				Expect(fileContents).To(Equal(`export CONTRAST__API__API_KEY=sample_api_key
export CONTRAST__API__URL=sample_teamserver_url/Contrast/
export CONTRAST__API__SERVICE_KEY=sample_service_key
export CONTRAST__API__USER_NAME=username@example.com
`))
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

				files, err := ioutil.ReadDir(path.Join(stager.BuildDir()))
				Expect(err).To(BeNil())
				Expect(len(files)).To(Equal(0))
			})

		})
	})

	Describe("GetCredentialsFromEnvironment", func() {

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
